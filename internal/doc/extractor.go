package doc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractResult contains the result of a document extraction
type ExtractResult struct {
	Path     string
	Text     string
	Title    string
	Metadata map[string]string
	Error    error
}

// SupportedFormats returns a list of supported document formats
func SupportedFormats() []string {
	return []string{
		".pdf", ".docx", ".doc", ".rtf", ".odt",
		".pptx", ".ppt", ".xlsx", ".xls", ".csv",
		".epub", ".html", ".htm", ".xml", ".txt",
	}
}

// IsSupported checks if a file format is supported for text extraction
func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, format := range SupportedFormats() {
		if ext == format {
			return true
		}
	}
	return false
}

// ExtractText extracts text from a document using the best available method
func ExtractText(ctx context.Context, filePath string) (*ExtractResult, error) {
	if !IsSupported(filePath) {
		return nil, fmt.Errorf("unsupported file format: %s", filepath.Ext(filePath))
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Determine the best extraction method based on file type
	ext := strings.ToLower(filepath.Ext(filePath))

	var text string
	var metadata map[string]string
	var title string
	var err error

	switch {
	case ext == ".pdf":
		text, metadata, err = extractPDF(ctx, filePath)
	case ext == ".docx" || ext == ".doc" || ext == ".odt" || ext == ".rtf":
		text, metadata, err = extractOfficeDocument(ctx, filePath)
	case ext == ".xlsx" || ext == ".xls" || ext == ".csv":
		text, metadata, err = extractSpreadsheet(ctx, filePath)
	case ext == ".pptx" || ext == ".ppt":
		text, metadata, err = extractPresentation(ctx, filePath)
	case ext == ".epub":
		text, metadata, err = extractEPUB(ctx, filePath)
	case ext == ".html" || ext == ".htm" || ext == ".xml":
		text, metadata, err = extractHTML(ctx, filePath)
	case ext == ".txt":
		text, err = extractTextFile(filePath)
		metadata = make(map[string]string)
	default:
		return nil, fmt.Errorf("no extraction method for format: %s", ext)
	}

	if err != nil {
		return &ExtractResult{
			Path:  filePath,
			Error: err,
		}, nil
	}

	// Extract title from metadata if available
	if t, ok := metadata["title"]; ok && t != "" {
		title = t
	} else {
		// Fallback to filename
		title = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	return &ExtractResult{
		Path:     filePath,
		Text:     text,
		Title:    title,
		Metadata: metadata,
	}, nil
}

// extractPDF extracts text and metadata from a PDF file
func extractPDF(ctx context.Context, path string) (string, map[string]string, error) {
	// Try pdftotext first (from poppler-utils)
	if _, err := exec.LookPath("pdftotext"); err == nil {
		cmd := exec.CommandContext(ctx, "pdftotext", "-enc", "UTF-8", path, "-")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("pdftotext failed: %w", err)
		}

		// Extract metadata with pdfinfo
		metadata, _ := extractPDFMetadata(ctx, path)
		return out.String(), metadata, nil
	}

	// Fallback to pdf2text if available
	if _, err := exec.LookPath("pdf2text"); err == nil {
		cmd := exec.CommandContext(ctx, "pdf2text", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("pdf2text failed: %w", err)
		}
		return out.String(), make(map[string]string), nil
	}

	return "", nil, fmt.Errorf("no PDF extraction tools available")
}

// extractPDFMetadata extracts metadata from a PDF using pdfinfo
func extractPDFMetadata(ctx context.Context, path string) (map[string]string, error) {
	metadata := make(map[string]string)

	if _, err := exec.LookPath("pdfinfo"); err != nil {
		return metadata, nil // Not an error, just return empty metadata
	}

	cmd := exec.CommandContext(ctx, "pdfinfo", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return metadata, fmt.Errorf("pdfinfo failed: %w", err)
	}

	// Parse pdfinfo output
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			value := strings.TrimSpace(parts[1])
			metadata[key] = value
		}
	}

	return metadata, nil
}

// extractOfficeDocument extracts text from Microsoft Office/LibreOffice documents
func extractOfficeDocument(ctx context.Context, path string) (string, map[string]string, error) {
	// Try Apache Tika if available
	if _, err := exec.LookPath("tika"); err == nil {
		// Use Tika for both text and metadata
		cmd := exec.CommandContext(ctx, "tika", "--text", "--encoding=UTF-8", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("tika failed: %w", err)
		}

		// Get metadata with tika
		metadata, _ := extractTikaMetadata(ctx, path)
		return out.String(), metadata, nil
	}

	// Try pandoc as fallback
	if _, err := exec.LookPath("pandoc"); err == nil {
		cmd := exec.CommandContext(ctx, "pandoc", "-t", "plain", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("pandoc failed: %w", err)
		}
		return out.String(), make(map[string]string), nil
	}

	// Try textutil on macOS
	if _, err := exec.LookPath("textutil"); err == nil {
		cmd := exec.CommandContext(ctx, "textutil", "-convert", "txt", "-stdout", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("textutil failed: %w", err)
		}
		return out.String(), make(map[string]string), nil
	}

	return "", nil, fmt.Errorf("no Office document extraction tools available")
}

// extractSpreadsheet extracts text from spreadsheet files
func extractSpreadsheet(ctx context.Context, path string) (string, map[string]string, error) {
	// Try Apache Tika for best results
	if _, err := exec.LookPath("tika"); err == nil {
		cmd := exec.CommandContext(ctx, "tika", "--text", "--encoding=UTF-8", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("tika failed: %w", err)
		}

		// Get metadata with tika
		metadata, _ := extractTikaMetadata(ctx, path)
		return out.String(), metadata, nil
	}

	// Try pandas in Python for CSV/Excel files
	if _, err := exec.LookPath("python3"); err == nil {
		// Create a temporary Python script
		tempScript := filepath.Join(os.TempDir(), "extract_excel.py")
		script := `
import sys
import pandas as pd

try:
    df = pd.read_excel(sys.argv[1]) if not sys.argv[1].endswith('.csv') else pd.read_csv(sys.argv[1])
    print(df.to_string())
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
`
		if err := os.WriteFile(tempScript, []byte(script), 0644); err != nil {
			return "", nil, fmt.Errorf("failed to create Python script: %w", err)
		}
		defer os.Remove(tempScript)

		cmd := exec.CommandContext(ctx, "python3", tempScript, path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("pandas extraction failed: %w", err)
		}
		return out.String(), make(map[string]string), nil
	}

	return "", nil, fmt.Errorf("no spreadsheet extraction tools available")
}

// extractPresentation extracts text from presentation files
func extractPresentation(ctx context.Context, path string) (string, map[string]string, error) {
	// Try Apache Tika
	if _, err := exec.LookPath("tika"); err == nil {
		cmd := exec.CommandContext(ctx, "tika", "--text", "--encoding=UTF-8", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("tika failed: %w", err)
		}

		// Get metadata with tika
		metadata, _ := extractTikaMetadata(ctx, path)
		return out.String(), metadata, nil
	}

	// Try pandoc as fallback
	if _, err := exec.LookPath("pandoc"); err == nil {
		cmd := exec.CommandContext(ctx, "pandoc", "-t", "plain", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("pandoc failed: %w", err)
		}
		return out.String(), make(map[string]string), nil
	}

	return "", nil, fmt.Errorf("no presentation extraction tools available")
}

// extractEPUB extracts text from EPUB files
func extractEPUB(ctx context.Context, path string) (string, map[string]string, error) {
	// Try pandoc
	if _, err := exec.LookPath("pandoc"); err == nil {
		cmd := exec.CommandContext(ctx, "pandoc", "-t", "plain", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("pandoc failed: %w", err)
		}
		return out.String(), make(map[string]string), nil
	}

	// Try Apache Tika
	if _, err := exec.LookPath("tika"); err == nil {
		cmd := exec.CommandContext(ctx, "tika", "--text", "--encoding=UTF-8", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("tika failed: %w", err)
		}

		// Get metadata with tika
		metadata, _ := extractTikaMetadata(ctx, path)
		return out.String(), metadata, nil
	}

	return "", nil, fmt.Errorf("no EPUB extraction tools available")
}

// extractHTML extracts text from HTML/XML files
func extractHTML(ctx context.Context, path string) (string, map[string]string, error) {
	// Try html2text
	if _, err := exec.LookPath("html2text"); err == nil {
		cmd := exec.CommandContext(ctx, "html2text", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("html2text failed: %w", err)
		}
		return out.String(), make(map[string]string), nil
	}

	// Try Apache Tika
	if _, err := exec.LookPath("tika"); err == nil {
		cmd := exec.CommandContext(ctx, "tika", "--text", "--encoding=UTF-8", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("tika failed: %w", err)
		}

		// Get metadata with tika
		metadata, _ := extractTikaMetadata(ctx, path)
		return out.String(), metadata, nil
	}

	// Read the file directly as fallback
	content, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read HTML file: %w", err)
	}

	return string(content), make(map[string]string), nil
}

// extractTextFile extracts text from plain text files
func extractTextFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read text file: %w", err)
	}

	return string(content), nil
}

// extractTikaMetadata extracts metadata using Apache Tika
func extractTikaMetadata(ctx context.Context, path string) (map[string]string, error) {
	metadata := make(map[string]string)

	if _, err := exec.LookPath("tika"); err != nil {
		return metadata, nil // Not an error, just return empty metadata
	}

	cmd := exec.CommandContext(ctx, "tika", "--metadata", "--json", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return metadata, fmt.Errorf("tika metadata extraction failed: %w", err)
	}

	// Simple parsing of key-value pairs (without proper JSON parsing)
	output := out.String()
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "\"") || !strings.Contains(line, "\":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.Trim(parts[0], "\" ")
		value := strings.Trim(parts[1], "\" ,")
		if key != "" && value != "" {
			metadata[strings.ToLower(key)] = value
		}
	}

	return metadata, nil
}
