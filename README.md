# Nikon RAW to JPEG Converter

A high-performance Go application for converting Nikon NEF (RAW) files to enhanced JPEG images using dcraw and advanced image processing.

## Features

- ✨ **High-Quality Conversion**: Uses dcraw's AHD interpolation algorithm
- 🎨 **Automatic Enhancement**: Applies contrast, brightness, saturation, and sharpening
- ⚡ **Fast Processing**: Efficient Go implementation
- 🔥 **Wildcard Support**: Convert multiple files with `*.NEF` patterns
- 🖼️ **Professional Output**: 95% JPEG quality with optimized color space
- 🔧 **Cross-Platform**: Works on Linux, macOS, and Windows

## Requirements

### System Dependencies

- **Go**: 1.16 or later
- **dcraw**: RAW image decoder
  - Ubuntu/Debian: `sudo apt install dcraw`
  - macOS: `brew install dcraw`
  - Arch: `sudo pacman -S dcraw`

### Go Dependencies

- `github.com/disintegration/imaging` - Image processing library
- `golang.org/x/image/tiff` - TIFF format support

## Installation

### Quick Install

```bash
# Initialize Go module (first time only)
go mod init nikonraw

# Install Go dependencies
go get github.com/disintegration/imaging
go get golang.org/x/image/tiff
go mod tidy

# Build the application
go build -o nikonraw main-wildcard.go

# Optional: Install system-wide
sudo mv nikonraw /usr/local/bin/
```

### One-Line Install

```bash
go mod init nikonraw && go get github.com/disintegration/imaging && go get golang.org/x/image/tiff && go mod tidy && go build -o nikonraw main-wildcard.go
```

## Usage

### Single File Conversion

Convert a single NEF file to JPEG:

```bash
./nikonraw input.NEF output.jpg
```

**Example:**
```bash
./nikonraw DSC_0031.NEF output.jpg
```

**Output:**
```
Converting RAW file...
Input:  DSC_0031.NEF
Output: output.jpg

Decoding RAW file with dcraw...
  Input: /full/path/to/DSC_0031.NEF
  Temp TIFF: /tmp/DSC_0031_1729512345678.tiff
Loading Nikon D750 image from DSC_0031.NEF ...
Scaling with darkness 600, saturation 16383, and
multipliers 2.019531 1.000000 1.355469 1.000000
AHD interpolation...
Converting to sRGB colorspace...
  TIFF created: 70.00 MB
Loading TIFF image...
Applying image enhancements...
Saving JPEG...

✓ Conversion successful!
Output file size: 18.34 MB
```

### Batch Conversion with Wildcards

Convert multiple files using wildcards:

```bash
# Convert all NEF files in current directory
./nikonraw "*.NEF" converted/

# Convert specific pattern
./nikonraw "DSC_*.NEF" output/

# Convert from subdirectory
./nikonraw "photos/*.NEF" processed/
```

**Important**: Always use quotes around wildcards to prevent shell expansion!

**Batch Output:**
```
==========================================
  Batch RAW to JPEG Converter
==========================================
Files to process: 15
Output directory: converted/
==========================================

[1/15 - 7%] Converting: DSC_0031.NEF ✓ (18.3 MB)
[2/15 - 13%] Converting: DSC_0032.NEF ✓ (16.7 MB)
[3/15 - 20%] Converting: DSC_0033.NEF ✓ (19.2 MB)
[4/15 - 27%] Converting: DSC_0034.NEF ✓ (17.9 MB)
[5/15 - 33%] Converting: DSC_0035.NEF ✓ (20.1 MB)
...

==========================================
  Conversion Summary
==========================================
✓ Successful: 15
Total:        15
Time elapsed: 2m 45s
Average:      11s per file
==========================================
```

## Image Processing Pipeline

The converter applies the following enhancements automatically:

1. **RAW Decoding** (dcraw)
   - AHD interpolation (highest quality)
   - Camera white balance
   - sRGB color space conversion

2. **Enhancement Filters** (Go imaging)
   - **Contrast**: +10-12% (platform-optimized)
   - **Brightness**: +5%
   - **Saturation**: +30%
   - **Sharpening**: 1.0 sigma
   - **Gamma**: 1.1 adjustment

3. **JPEG Encoding**
   - 95% quality setting
   - Optimized compression

## Technical Details

### dcraw Parameters

- `-T`: Output TIFF format (16-bit)
- `-w`: Use camera white balance
- `-q 3`: High-quality AHD interpolation
- `-c`: Write to stdout (allows redirection to `/tmp/`)

### File Handling

The converter intelligently handles dcraw's output to avoid space issues:

1. dcraw outputs to stdout using `-c` flag
2. Temporary TIFF file is written to `/tmp/` directory (not source location)
3. This prevents filling up SD cards or external media with large temp files
4. Temporary TIFF file is automatically cleaned up after conversion
5. Output JPEG is saved to the specified location
6. Preserves original NEF files (read-only operation)

**Why /tmp/?** 
- Avoids "No space left on device" errors when reading from SD cards
- System temp directory typically has more available space
- Automatic cleanup even if process is interrupted

### Supported Cameras

Works with all Nikon cameras that produce NEF files, including:
- Nikon D750, D780, D850, D500
- Nikon Z6, Z7, Z6 II, Z7 II, Z8, Z9
- Nikon D3xxx, D5xxx, D7xxx series
- And many more (any camera supported by dcraw)

## Examples

### Basic Conversion

```bash
# Single file
./nikonraw photo.NEF photo.jpg

# With absolute paths
./nikonraw /home/user/Photos/DSC_0031.NEF /home/user/Converted/image.jpg
```

### Wildcard Operations

```bash
# Convert all NEF files in current directory
./nikonraw "*.NEF" converted/

# Convert specific pattern (all DSC files)
./nikonraw "DSC_*.NEF" output/

# Convert files with specific numbers
./nikonraw "DSC_003?.NEF" selected/

# Convert from subdirectory
./nikonraw "raw-photos/*.NEF" processed/
```

**Important**: Always use quotes around wildcards to prevent shell expansion!

### Integration with Scripts

```bash
# Process latest photo
LATEST=$(ls -t *.NEF | head -1)
./nikonraw "$LATEST" "latest.jpg"

# Automated workflow
for file in *.NEF; do
    output="processed/${file%.NEF}.jpg"
    ./nikonraw "$file" "$output"
done
```

## Troubleshooting

### "dcraw not found"

**Problem**: The dcraw utility is not installed.

**Solution**:
```bash
# Ubuntu/Debian
sudo apt install dcraw

# macOS
brew install dcraw

# Arch Linux
sudo pacman -S dcraw
```

### "failed to open TIFF"

**Problem**: dcraw failed to create the intermediate TIFF file.

**Solutions**:
1. Check if input NEF file is valid
2. Ensure sufficient space in `/tmp/` (TIFF files are ~70MB each)
3. Verify read permissions on input file and write permissions on `/tmp/`

### "No space left on device"

**Problem**: Not enough space in `/tmp/` for temporary TIFF file.

**Solutions**:
```bash
# Check /tmp/ space
df -h /tmp

# Clear /tmp/ if needed
sudo rm -rf /tmp/DSC_*.tiff

# Or set TMPDIR to a location with more space
export TMPDIR=/path/to/large/drive/tmp
mkdir -p $TMPDIR
```

### "No NEF files found"

**Problem**: Wildcard pattern didn't match any files.

**Solutions**:
1. Check your wildcard pattern
2. Verify files exist: `ls *.NEF`
3. Make sure you used quotes around the pattern

### Permission Denied

**Problem**: Cannot write output file or create temporary TIFF.

**Solution**:
```bash
# Ensure write permissions
chmod +w .

# Or specify output directory where you have permissions
./nikonraw input.NEF ~/Desktop/output.jpg
```

## Performance

**Typical conversion times** (on modern hardware):
- Single 24MP NEF file: ~10-15 seconds
- Batch of 100 files: ~20-25 minutes
- Processing is I/O bound (disk speed matters)

**File sizes**:
- Input NEF: ~25-30 MB
- Intermediate TIFF: ~70 MB (temporary)
- Output JPEG (95% quality): ~15-20 MB

## Building from Source

### Standard Build

```bash
go build -o nikonraw main-wildcard.go
```

### Optimized Build

```bash
# Smaller binary with optimizations
go build -ldflags="-s -w" -o nikonraw main-wildcard.go

# Further compress with upx (optional)
upx --best --lzma nikonraw
```

## Project Structure

```
nikonraw-converter/
├── main-wildcard.go    # Main application with wildcard support
├── README.md           # This file
├── go.mod              # Go module definition (created by go mod init)
└── go.sum              # Go dependency checksums (created automatically)
```

**Note**: `go.mod` and `go.sum` are created automatically when you run `go mod init` and `go mod tidy`.

## Contributing

Contributions are welcome! Areas for improvement:

- [ ] Add support for other RAW formats (CR2, ARW, etc.)
- [ ] Implement concurrent batch processing
- [ ] Add GUI interface
- [ ] Custom enhancement profiles
- [ ] EXIF data preservation
- [ ] Progress bars for large files

## License

This project is provided as-is for personal and commercial use.

## Credits

- **dcraw**: Dave Coffin's universal RAW converter
- **imaging**: Disintegration's Go imaging library
- Built with ❤️ for Nikon photographers

## Changelog

### v1.2 (Latest)
- 🔥 **CRITICAL FIX**: Use `/tmp/` for temporary TIFF files instead of source directory
- ✅ Prevents "No space left on device" errors when reading from SD cards
- ✅ Added timestamp to temp filenames to avoid conflicts
- ✅ Uses dcraw's `-c` flag to write to stdout

### v1.1
- ✨ Built-in wildcard support (`*.NEF`) for batch conversion
- ✨ Automatic batch mode detection
- ✅ Quiet mode for batch operations
- ✅ Progress indicators with file counts and percentages
- ✅ Comprehensive conversion summaries

### v1.0
- ✅ Fixed dcraw file location handling
- ✅ Automatic TIFF cleanup
- ✅ Enhanced error messages
- ✅ Cross-platform support

---

**Questions or issues?** Open an issue or submit a pull request!

**Happy shooting! 📸**
