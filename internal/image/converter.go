package image

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertOptions contains options for image conversion
type ConvertOptions struct {
	SourcePath   string
	OutputPath   string
	OutputFormat string
	Quality      int
}

// ConvertResult represents the result of an image conversion
type ConvertResult struct {
	InputPath  string
	OutputPath string
	SizeBytes  int64
	Error      error
}

// DefaultOptions returns default conversion options
func DefaultOptions() ConvertOptions {
	return ConvertOptions{
		OutputFormat: "jpg",
		Quality:      85,
	}
}

// Convert converts an image file to the specified format
func Convert(ctx context.Context, options ConvertOptions) (*ConvertResult, error) {
	if options.SourcePath == "" {
		return nil, fmt.Errorf("source path is required")
	}

	// Check if source file exists
	if _, err := os.Stat(options.SourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("source file does not exist: %s", options.SourcePath)
	}

	// Generate output path if not provided
	if options.OutputPath == "" {
		dir := filepath.Dir(options.SourcePath)
		filename := filepath.Base(options.SourcePath)
		ext := filepath.Ext(filename)
		basename := strings.TrimSuffix(filename, ext)
		options.OutputPath = filepath.Join(dir, basename+"."+options.OutputFormat)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(options.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine conversion tool and arguments based on input format
	var cmd *exec.Cmd

	ext := strings.ToLower(filepath.Ext(options.SourcePath))
	if ext == ".heic" || ext == ".heif" {
		// Use sips for HEIC conversion on macOS
		if _, err := exec.LookPath("sips"); err == nil {
			cmd = exec.CommandContext(ctx, "sips",
				"-s", "format", options.OutputFormat,
				"-s", "formatOptions", fmt.Sprintf("normal %d", options.Quality),
				options.SourcePath,
				"--out", options.OutputPath,
			)
		} else {
			// Fallback to ImageMagick if available
			if _, err := exec.LookPath("convert"); err == nil {
				cmd = exec.CommandContext(ctx, "convert",
					options.SourcePath,
					"-quality", fmt.Sprintf("%d", options.Quality),
					options.OutputPath,
				)
			} else {
				return nil, fmt.Errorf("no suitable conversion tool found for HEIC format")
			}
		}
	} else if ext == ".avif" {
		// Check for ImageMagick
		if _, err := exec.LookPath("convert"); err == nil {
			cmd = exec.CommandContext(ctx, "convert",
				options.SourcePath,
				"-quality", fmt.Sprintf("%d", options.Quality),
				options.OutputPath,
			)
		} else {
			return nil, fmt.Errorf("no suitable conversion tool found for AVIF format")
		}
	} else {
		// Use ffmpeg for all other formats as it's more widely available
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-y",
			"-i", options.SourcePath,
			"-q:v", fmt.Sprintf("%d", 100-options.Quality), // ffmpeg quality is inverse (1-31)
			options.OutputPath,
		)
	}

	// Run the conversion command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ConvertResult{
			InputPath:  options.SourcePath,
			OutputPath: options.OutputPath,
			Error:      fmt.Errorf("conversion failed: %w\nOutput: %s", err, string(output)),
		}, nil
	}

	// Get info about the output file
	fileInfo, err := os.Stat(options.OutputPath)
	if err != nil {
		return &ConvertResult{
			InputPath:  options.SourcePath,
			OutputPath: options.OutputPath,
			Error:      fmt.Errorf("failed to get output file info: %w", err),
		}, nil
	}

	return &ConvertResult{
		InputPath:  options.SourcePath,
		OutputPath: options.OutputPath,
		SizeBytes:  fileInfo.Size(),
	}, nil
}

// IsHEIC checks if a file is in HEIC format
func IsHEIC(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".heic" || ext == ".heif"
}

// IsAVIF checks if a file is in AVIF format
func IsAVIF(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".avif"
}

// IsSupportedInputFormat checks if a file format is supported for conversion
func IsSupportedInputFormat(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	supportedExts := []string{
		".heic", ".heif", ".avif",
		".jpg", ".jpeg", ".png",
		".tiff", ".tif", ".raw",
		".cr2", ".nef", ".arw",
	}

	for _, supported := range supportedExts {
		if ext == supported {
			return true
		}
	}

	return false
}
