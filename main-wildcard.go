package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/tiff"
)

// ConvertRAWToJPEG converts RAW to JPEG using dcraw and pure Go processing
func ConvertRAWToJPEG(inputPath, outputPath string, quiet bool) error {
	// Ensure output has .jpg extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".jpg") &&
		!strings.HasSuffix(strings.ToLower(outputPath), ".jpeg") {
		outputPath += ".jpg"
	}

	// Check if dcraw is available
	dcrawPath, err := exec.LookPath("dcraw")
	if err != nil {
		return fmt.Errorf("dcraw not found. Install with: sudo apt install dcraw")
	}

	// Get absolute path of input
	absInputPath, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Use /tmp/ for temporary TIFF to avoid filling up source device (e.g., SD card)
	inputBase := filepath.Base(absInputPath)
	inputNoExt := strings.TrimSuffix(inputBase, filepath.Ext(inputBase))
	
	// Create unique temp filename with timestamp to avoid conflicts
	timestamp := time.Now().UnixNano()
	tempTiffName := fmt.Sprintf("%s_%d.tiff", inputNoExt, timestamp)
	dcrawOutput := filepath.Join("/tmp", tempTiffName)

	// Remove any existing .tiff file
	os.Remove(dcrawOutput)

	if !quiet {
		fmt.Println("Decoding RAW file with dcraw...")
		fmt.Printf("  Input: %s\n", absInputPath)
		fmt.Printf("  Temp TIFF: %s\n", dcrawOutput)
	}
	
	// Run dcraw with TIFF output
	// -T: output as TIFF
	// -w: use camera white balance
	// -q 3: high quality interpolation (AHD)
	// -c: write to stdout, then we redirect to /tmp/
	cmd := exec.Command(dcrawPath, "-T", "-w", "-q", "3", "-c", absInputPath)
	
	// Create output file in /tmp/
	outFile, err := os.Create(dcrawOutput)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer outFile.Close()
	
	cmd.Stdout = outFile
	
	if !quiet {
		cmd.Stderr = os.Stderr
	}
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dcraw failed: %v", err)
	}

	// Check if dcraw created the output file
	if _, err := os.Stat(dcrawOutput); os.IsNotExist(err) {
		return fmt.Errorf("dcraw did not create expected output: %s", dcrawOutput)
	}

	// Get file size
	fileInfo, err := os.Stat(dcrawOutput)
	if err != nil {
		return fmt.Errorf("cannot stat dcraw output: %v", err)
	}
	
	if !quiet {
		fmt.Printf("  TIFF created: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))
	}

	// Load the TIFF image
	if !quiet {
		fmt.Println("Loading TIFF image...")
	}
	img, err := imaging.Open(dcrawOutput)
	if err != nil {
		return fmt.Errorf("failed to open TIFF: %v", err)
	}

	// Clean up TIFF file
	defer os.Remove(dcrawOutput)

	if !quiet {
		fmt.Println("Applying image enhancements...")
	}
	img = processImage(img)

	if !quiet {
		fmt.Println("Saving JPEG...")
	}
	err = imaging.Save(img, outputPath, imaging.JPEGQuality(95))
	if err != nil {
		return fmt.Errorf("failed to save JPEG: %v", err)
	}

	return nil
}

// processImage applies enhancement filters to the image
func processImage(img image.Image) image.Image {
	contrastAmount := 10.0
	if runtime.GOOS == "linux" {
		contrastAmount = 12.0
	}
	img = imaging.AdjustContrast(img, contrastAmount)
	img = imaging.AdjustBrightness(img, 5.0)
	img = imaging.AdjustSaturation(img, 30.0)
	img = imaging.Sharpen(img, 1.0)
	img = imaging.AdjustGamma(img, 1.1)
	return img
}

func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// findNEFFiles finds all NEF files matching the pattern
func findNEFFiles(pattern string) ([]string, error) {
	// If pattern contains wildcards, use glob
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern: %v", err)
		}
		
		// Filter to only .NEF files (case insensitive)
		var nefFiles []string
		for _, match := range matches {
			ext := strings.ToLower(filepath.Ext(match))
			if ext == ".nef" {
				nefFiles = append(nefFiles, match)
			}
		}
		return nefFiles, nil
	}
	
	// Single file
	return []string{pattern}, nil
}

// JobResult holds the result of a conversion job
type JobResult struct {
	filename string
	output   string
	err      error
	size     int64
	index    int
}

// convertBatch processes multiple files with parallel processing
func convertBatch(files []string, outputDir string, workers int) {
	total := len(files)
	startTime := time.Now()

	fmt.Println("==========================================")
	fmt.Println("  Batch RAW to JPEG Converter")
	fmt.Println("==========================================")
	fmt.Printf("Files to process: %d\n", total)
	fmt.Printf("Output directory: %s\n", outputDir)
	fmt.Printf("Parallel workers: %d\n", workers)
	fmt.Println("==========================================")
	fmt.Println()

	// Create output directory if needed
	if outputDir != "" {
		os.MkdirAll(outputDir, 0755)
	}

	// Create channels for work distribution
	jobs := make(chan struct {
		file  string
		index int
	}, total)
	results := make(chan JobResult, total)

	// Start worker goroutines
	for w := 0; w < workers; w++ {
		go func() {
			for job := range jobs {
				filename := filepath.Base(job.file)
				base := strings.TrimSuffix(filename, filepath.Ext(filename))
				
				var output string
				if outputDir != "" {
					output = filepath.Join(outputDir, base+".jpg")
				} else {
					output = base + ".jpg"
				}

				// Convert the file
				err := ConvertRAWToJPEG(job.file, output, true)
				
				result := JobResult{
					filename: filename,
					output:   output,
					err:      err,
					index:    job.index,
				}
				
				if err == nil {
					if size, sizeErr := GetFileSize(output); sizeErr == nil {
						result.size = size
					}
				}
				
				results <- result
			}
		}()
	}

	// Send jobs to workers
	for i, file := range files {
		jobs <- struct {
			file  string
			index int
		}{file, i}
	}
	close(jobs)

	// Collect and display results
	success := 0
	failed := 0
	processed := 0

	for processed < total {
		result := <-results
		processed++
		
		percentage := processed * 100 / total
		fmt.Printf("[%d/%d - %d%%] %s", processed, total, percentage, result.filename)

		if result.err != nil {
			failed++
			fmt.Printf(" ✗ Failed: %v\n", result.err)
		} else {
			success++
			if result.size > 0 {
				fmt.Printf(" ✓ (%.1f MB)\n", float64(result.size)/(1024*1024))
			} else {
				fmt.Println(" ✓")
			}
		}
	}

	// Summary
	elapsed := time.Since(startTime)
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  Conversion Summary")
	fmt.Println("==========================================")
	fmt.Printf("✓ Successful: %d\n", success)
	if failed > 0 {
		fmt.Printf("✗ Failed:     %d\n", failed)
	}
	fmt.Printf("Total:        %d\n", total)
	fmt.Printf("Workers:      %d\n", workers)
	fmt.Printf("Time elapsed: %s\n", formatDuration(elapsed))
	if success > 0 {
		avgTime := elapsed / time.Duration(success)
		fmt.Printf("Average:      %s per file\n", formatDuration(avgTime))
		
		// Show speedup compared to sequential
		seqTime := avgTime * time.Duration(success)
		speedup := float64(seqTime) / float64(elapsed)
		fmt.Printf("Speedup:      %.1fx faster than sequential\n", speedup)
	}
	fmt.Println("==========================================")
}

// formatDuration formats a duration nicely
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func printUsage() {
	fmt.Println("Nikon RAW to JPEG Converter")
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU Cores: %d\n\n", runtime.NumCPU())
	fmt.Println("Usage:")
	fmt.Println("  Single file:  nikonraw <input.nef> <output.jpg>")
	fmt.Println("  Wildcard:     nikonraw [-j N] <pattern> <output_directory>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -j N          Number of parallel workers (default: CPU cores)")
	fmt.Println("  -h, --help    Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  nikonraw DSC_0031.NEF output.jpg           # Convert single file")
	fmt.Println("  nikonraw \"*.NEF\" converted/                # Convert all (auto parallel)")
	fmt.Println("  nikonraw -j 4 \"*.NEF\" converted/          # Use 4 workers")
	fmt.Println("  nikonraw -j 1 \"*.NEF\" converted/          # Sequential (no parallel)")
	fmt.Println("  nikonraw \"DSC_*.NEF\" processed/            # Convert matching files")
	fmt.Println("  nikonraw \"photos/*.NEF\" jpg/               # Convert from subdirectory")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  NIKONRAW_WORKERS=N   Set default number of workers")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Output directory is created automatically if it doesn't exist")
	fmt.Println("  - For batch conversion, output filenames match input basenames")
	fmt.Println("  - Wildcards: * (any chars), ? (single char)")
	fmt.Println("  - Parallel processing speeds up batch conversions significantly")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Handle help flags
	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		printUsage()
		os.Exit(0)
	}

	// Parse arguments
	workers := runtime.NumCPU() // Default to number of CPU cores
	argOffset := 1

	// Check for -j flag
	if len(os.Args) >= 4 && os.Args[1] == "-j" {
		if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
			workers = n
			argOffset = 3
		} else {
			fmt.Fprintf(os.Stderr, "Error: Invalid worker count: %s\n", os.Args[2])
			os.Exit(1)
		}
	}

	// Check for environment variable
	if envWorkers := os.Getenv("NIKONRAW_WORKERS"); envWorkers != "" && argOffset == 1 {
		if n, err := strconv.Atoi(envWorkers); err == nil && n > 0 {
			workers = n
		}
	}

	// Need at least 2 more arguments after flags
	if len(os.Args) < argOffset+2 {
		printUsage()
		os.Exit(1)
	}

	inputPattern := os.Args[argOffset]
	outputPath := os.Args[argOffset+1]

	// Find matching files
	files, err := findNEFFiles(inputPattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No NEF files found matching: %s\n", inputPattern)
		os.Exit(1)
	}

	// Check if this is batch mode (multiple files or wildcard pattern)
	isBatch := len(files) > 1 || strings.Contains(inputPattern, "*") || strings.Contains(inputPattern, "?")

	if isBatch {
		// Batch mode: treat outputPath as directory
		// Limit workers to number of files if fewer files than workers
		if workers > len(files) {
			workers = len(files)
		}
		convertBatch(files, outputPath, workers)
	} else {
		// Single file mode
		inputFile := files[0]
		
		// Check if input file exists
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Input file does not exist: %s\n", inputFile)
			os.Exit(1)
		}

		fmt.Println("Converting RAW file...")
		fmt.Printf("Input:  %s\n", inputFile)
		fmt.Printf("Output: %s\n\n", outputPath)

		err := ConvertRAWToJPEG(inputFile, outputPath, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n✗ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n✓ Conversion successful!")

		if size, err := GetFileSize(outputPath); err == nil {
			fmt.Printf("Output file size: %.2f MB\n", float64(size)/(1024.0*1024.0))
		}
	}
}
