package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// convertBatch processes multiple files
func convertBatch(files []string, outputDir string) {
	total := len(files)
	success := 0
	failed := 0
	startTime := time.Now()

	fmt.Println("==========================================")
	fmt.Println("  Batch RAW to JPEG Converter")
	fmt.Println("==========================================")
	fmt.Printf("Files to process: %d\n", total)
	fmt.Printf("Output directory: %s\n", outputDir)
	fmt.Println("==========================================")
	fmt.Println()

	// Create output directory if needed
	if outputDir != "" {
		os.MkdirAll(outputDir, 0755)
	}

	for i, file := range files {
		filename := filepath.Base(file)
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		
		var output string
		if outputDir != "" {
			output = filepath.Join(outputDir, base+".jpg")
		} else {
			output = base + ".jpg"
		}

		// Progress
		percentage := (i + 1) * 100 / total
		fmt.Printf("[%d/%d - %d%%] Converting: %s", i+1, total, percentage, filename)

		// Convert
		err := ConvertRAWToJPEG(file, output, true)
		if err != nil {
			failed++
			fmt.Printf(" ✗ Failed: %v\n", err)
		} else {
			success++
			if size, err := GetFileSize(output); err == nil {
				fmt.Printf(" ✓ (%.1f MB)\n", float64(size)/(1024*1024))
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
	fmt.Printf("Time elapsed: %s\n", formatDuration(elapsed))
	if success > 0 {
		avgTime := elapsed / time.Duration(success)
		fmt.Printf("Average:      %s per file\n", formatDuration(avgTime))
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
	fmt.Printf("Platform: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("Usage:")
	fmt.Println("  Single file:  nikonraw <input.nef> <output.jpg>")
	fmt.Println("  Wildcard:     nikonraw <pattern> <output_directory>")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  nikonraw DSC_0031.NEF output.jpg           # Convert single file")
	fmt.Println("  nikonraw *.NEF converted/                  # Convert all NEF files")
	fmt.Println("  nikonraw DSC_*.NEF processed/              # Convert matching files")
	fmt.Println("  nikonraw \"photos/*.NEF\" jpg/               # Convert from subdirectory")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Output directory is created automatically if it doesn't exist")
	fmt.Println("  - For batch conversion, output filenames match input basenames")
	fmt.Println("  - Wildcards: * (any chars), ? (single char)")
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

	if len(os.Args) != 3 {
		printUsage()
		os.Exit(1)
	}

	inputPattern := os.Args[1]
	outputPath := os.Args[2]

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
		convertBatch(files, outputPath)
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
