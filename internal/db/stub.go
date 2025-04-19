package db

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StubMode represents the format of the stub file
type StubMode string

const (
	// StubModeWebloc creates .webloc stubs (macOS)
	StubModeWebloc StubMode = "webloc"
	// StubModeShortcut creates .url stubs (Windows)
	StubModeShortcut StubMode = "shortcut"
	// StubModeNone doesn't create stubs
	StubModeNone StubMode = "none"
)

// StubResult represents the result of creating a stub
type StubResult struct {
	OriginalPath string
	StubPath     string
	URL          string
	Mode         StubMode
	Error        error
}

// WeblocFile represents an Apple .webloc file
type WeblocFile struct {
	XMLName xml.Name `xml:"plist"`
	Version string   `xml:"version,attr"`
	Dict    struct {
		Key   string `xml:"key"`
		Value string `xml:"string"`
	} `xml:"dict"`
}

// CreateStub creates a stub file that points to a URL
func CreateStub(originalPath, url string, mode StubMode) (*StubResult, error) {
	result := &StubResult{
		OriginalPath: originalPath,
		URL:          url,
		Mode:         mode,
	}

	// No-op if mode is none
	if mode == StubModeNone {
		return result, nil
	}

	// Determine stub path
	var stubPath string
	switch mode {
	case StubModeWebloc:
		stubPath = originalPath + ".webloc"
	case StubModeShortcut:
		stubPath = originalPath + ".url"
	default:
		return nil, fmt.Errorf("unsupported stub mode: %s", mode)
	}
	result.StubPath = stubPath

	var err error
	switch mode {
	case StubModeWebloc:
		err = createWeblocFile(stubPath, url)
	case StubModeShortcut:
		err = createShortcutFile(stubPath, url)
	}

	if err != nil {
		result.Error = err
		return result, err
	}

	return result, nil
}

// ReplaceWithStub replaces a file with a stub
func ReplaceWithStub(originalPath, url string, mode StubMode) (*StubResult, error) {
	// Create the stub
	result, err := CreateStub(originalPath, url, mode)
	if err != nil {
		return result, err
	}

	// Skip if mode is none
	if mode == StubModeNone {
		return result, nil
	}

	// Remove the original file
	if err := os.Remove(originalPath); err != nil {
		result.Error = fmt.Errorf("failed to remove original file: %w", err)
		return result, result.Error
	}

	return result, nil
}

// CreateStubsForDirectory creates stubs for all files in a directory that have been uploaded
func CreateStubsForDirectory(db *DB, directory string, mode StubMode) (int, error) {
	if mode == StubModeNone {
		return 0, nil
	}

	// Get all files in the directory
	files, err := db.GetFilesInDirectory(directory)
	if err != nil {
		return 0, err
	}

	// Create stubs for files that have been uploaded
	count := 0
	var firstErr error
	for _, file := range files {
		if file.IsDir || file.UploadedURL == "" {
			continue
		}

		_, err := ReplaceWithStub(file.Path, file.UploadedURL, mode)
		if err != nil && firstErr == nil {
			firstErr = err
		} else {
			count++
		}
	}

	return count, firstErr
}

// createWeblocFile creates a .webloc file (macOS)
func createWeblocFile(path, url string) error {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the XML structure for a .webloc file
	webloc := WeblocFile{
		Version: "1.0",
	}
	webloc.Dict.Key = "URL"
	webloc.Dict.Value = url

	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write XML header
	file.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
`)

	// Encode the structure
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "    ")
	if err := encoder.Encode(webloc); err != nil {
		return fmt.Errorf("failed to encode XML: %w", err)
	}

	return nil
}

// createShortcutFile creates a .url file (Windows)
func createShortcutFile(path, url string) error {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write the contents
	content := fmt.Sprintf(`[InternetShortcut]
URL=%s
`, url)
	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// GetFilesInDirectory gets all files in a directory from the database
func (db *DB) GetFilesInDirectory(directory string) ([]*FileStatus, error) {
	query := `
	SELECT id, path, relative_path, size, mod_time, is_dir, content_type, 
	       sha256, processed, uploaded_url, upload_time, summary
	FROM files
	WHERE path LIKE ?
	ORDER BY path
	`

	// Add wildcard to match all files in the directory
	directoryPattern := directory
	if !strings.HasSuffix(directoryPattern, "/") {
		directoryPattern += "/"
	}
	directoryPattern += "%"

	rows, err := db.conn.Query(query, directoryPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*FileStatus
	for rows.Next() {
		var file FileStatus
		err := rows.Scan(
			&file.ID,
			&file.Path,
			&file.RelativePath,
			&file.Size,
			&file.ModTime,
			&file.IsDir,
			&file.ContentType,
			&file.SHA256,
			&file.Processed,
			&file.UploadedURL,
			&file.UploadTime,
			&file.Summary,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}
