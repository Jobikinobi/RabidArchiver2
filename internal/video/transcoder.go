package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// TranscodeOptions contains options for video transcoding
type TranscodeOptions struct {
	SourcePath       string
	OutputPath       string
	OutputFormat     string
	UseHardwareAccel bool
	Quality          string
}

// TranscodeResult represents the result of a transcoding operation
type TranscodeResult struct {
	InputPath       string
	OutputPath      string
	OutputFormat    string
	DurationSeconds float64
	SizeBytes       int64
	Error           error
}

// DefaultOptions returns default transcoding options
func DefaultOptions() TranscodeOptions {
	return TranscodeOptions{
		OutputFormat:     "mp4",
		UseHardwareAccel: true,
		Quality:          "medium",
	}
}

// Transcode transcodes a video file using ffmpeg
func Transcode(ctx context.Context, options TranscodeOptions) (*TranscodeResult, error) {
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
		options.OutputPath = filepath.Join(dir, basename+".transcoded."+options.OutputFormat)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(options.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build ffmpeg command
	args := buildFFmpegArgs(options)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Capture output for logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &TranscodeResult{
			InputPath:  options.SourcePath,
			OutputPath: options.OutputPath,
			Error:      fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, string(output)),
		}, nil
	}

	// Get info about the output file
	fileInfo, err := os.Stat(options.OutputPath)
	if err != nil {
		return &TranscodeResult{
			InputPath:  options.SourcePath,
			OutputPath: options.OutputPath,
			Error:      fmt.Errorf("failed to get output file info: %w", err),
		}, nil
	}

	// Get video duration
	duration, err := getVideoDuration(options.OutputPath)
	if err != nil {
		// Non-fatal error, just log it
		fmt.Printf("Warning: could not get video duration: %v\n", err)
	}

	return &TranscodeResult{
		InputPath:       options.SourcePath,
		OutputPath:      options.OutputPath,
		OutputFormat:    options.OutputFormat,
		DurationSeconds: duration,
		SizeBytes:       fileInfo.Size(),
	}, nil
}

// buildFFmpegArgs builds the ffmpeg command arguments based on options
func buildFFmpegArgs(options TranscodeOptions) []string {
	args := []string{
		"-i", options.SourcePath,
		"-y", // Overwrite output files without asking
	}

	// Add hardware acceleration if requested and available
	if options.UseHardwareAccel {
		if runtime.GOOS == "darwin" {
			// macOS VideoToolbox hardware acceleration
			args = append(args, "-c:v", "h264_videotoolbox")
		} else if runtime.GOOS == "linux" {
			// Try VAAPI on Linux if available
			args = append(args, "-vaapi_device", "/dev/dri/renderD128", "-vf", "format=nv12,hwupload", "-c:v", "h264_vaapi")
		}
	} else {
		// Software encoding
		args = append(args, "-c:v", "libx264")
	}

	// Set quality based on option
	switch options.Quality {
	case "low":
		args = append(args, "-preset", "ultrafast", "-crf", "28")
	case "medium":
		args = append(args, "-preset", "medium", "-crf", "23")
	case "high":
		args = append(args, "-preset", "slow", "-crf", "18")
	}

	// Copy audio codec
	args = append(args, "-c:a", "aac", "-b:a", "128k")

	// Add output file
	args = append(args, options.OutputPath)

	return args
}

// getVideoDuration gets the duration of a video file in seconds
func getVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var duration float64
	if _, err := fmt.Sscanf(string(output), "%f", &duration); err != nil {
		return 0, err
	}

	return duration, nil
}

// ExtractAudio extracts audio from a video file
func ExtractAudio(ctx context.Context, videoPath, outputPath string) error {
	if outputPath == "" {
		dir := filepath.Dir(videoPath)
		filename := filepath.Base(videoPath)
		ext := filepath.Ext(filename)
		basename := strings.TrimSuffix(filename, ext)
		outputPath = filepath.Join(dir, basename+".mp3")
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", videoPath,
		"-q:a", "0",
		"-map", "a",
		outputPath,
	)

	return cmd.Run()
}

// GenerateWhisperTranscript generates a transcript using Whisper
func GenerateWhisperTranscript(ctx context.Context, audioPath string) (string, error) {
	// Check if whisper exists
	_, err := exec.LookPath("whisper")
	if err != nil {
		return "", fmt.Errorf("whisper not found in PATH, cannot generate transcript")
	}

	outputDir := filepath.Dir(audioPath)
	baseFileName := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	outputTxtPath := filepath.Join(outputDir, baseFileName+".txt")

	// Run whisper with a timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "whisper",
		"--model", "tiny", // Use tiny model for speed
		"--output_format", "txt",
		"--output_dir", outputDir,
		audioPath,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper transcription failed: %w", err)
	}

	// Read the transcript
	transcript, err := os.ReadFile(outputTxtPath)
	if err != nil {
		return "", fmt.Errorf("failed to read transcript file: %w", err)
	}

	return string(transcript), nil
}
