package scan

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FileInfo represents metadata about a file
type FileInfo struct {
	Path         string
	RelativePath string
	Size         int64
	ModTime      time.Time
	IsDir        bool
	ContentType  string
	SHA256       string
}

// Scanner scans a directory and builds a manifest
type Scanner struct {
	db         *sql.DB
	sourcePath string
	dbPath     string
}

// NewScanner creates a new scanner
func NewScanner(sourcePath, dbPath string) (*Scanner, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	scanner := &Scanner{
		db:         db,
		sourcePath: sourcePath,
		dbPath:     dbPath,
	}

	if err := scanner.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return scanner, nil
}

// Close closes the database connection
func (s *Scanner) Close() error {
	return s.db.Close()
}

// initDB initializes the database schema
func (s *Scanner) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL,
		relative_path TEXT NOT NULL,
		size INTEGER NOT NULL,
		mod_time DATETIME NOT NULL,
		is_dir BOOLEAN NOT NULL,
		content_type TEXT,
		sha256 TEXT,
		processed BOOLEAN DEFAULT FALSE,
		uploaded_url TEXT,
		upload_time DATETIME,
		summary TEXT,
		UNIQUE(path)
	);
	CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
	CREATE INDEX IF NOT EXISTS idx_files_relative_path ON files(relative_path);
	CREATE INDEX IF NOT EXISTS idx_files_processed ON files(processed);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Scan scans the source directory and builds a manifest
func (s *Scanner) Scan() error {
	return filepath.Walk(s.sourcePath, s.processFile)
}

// processFile processes a single file or directory
func (s *Scanner) processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	// Skip hidden files and directories
	if strings.HasPrefix(filepath.Base(path), ".") {
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}

	relPath, err := filepath.Rel(s.sourcePath, path)
	if err != nil {
		return err
	}

	fileInfo := FileInfo{
		Path:         path,
		RelativePath: relPath,
		Size:         info.Size(),
		ModTime:      info.ModTime(),
		IsDir:        info.IsDir(),
	}

	if !info.IsDir() {
		contentType, err := detectContentType(path)
		if err != nil {
			return err
		}
		fileInfo.ContentType = contentType

		// Calculate hash for files smaller than 1GB
		if info.Size() < 1073741824 {
			hash, err := calculateSHA256(path)
			if err != nil {
				return err
			}
			fileInfo.SHA256 = hash
		}
	}

	return s.saveFileInfo(fileInfo)
}

// saveFileInfo saves file information to the database
func (s *Scanner) saveFileInfo(info FileInfo) error {
	query := `
	INSERT OR REPLACE INTO files 
	(path, relative_path, size, mod_time, is_dir, content_type, sha256)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		info.Path,
		info.RelativePath,
		info.Size,
		info.ModTime,
		info.IsDir,
		info.ContentType,
		info.SHA256,
	)

	return err
}

// detectContentType attempts to determine the MIME type of a file
func detectContentType(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	return detectMIMEType(buffer, filepath.Ext(path)), nil
}

// detectMIMEType detects MIME type based on file contents and extension
func detectMIMEType(buffer []byte, extension string) string {
	contentType := http.DetectContentType(buffer)

	// Map common extensions to MIME types if detection is generic
	if contentType == "application/octet-stream" {
		switch strings.ToLower(extension) {
		case ".pdf":
			return "application/pdf"
		case ".doc":
			return "application/msword"
		case ".docx":
			return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case ".xls":
			return "application/vnd.ms-excel"
		case ".xlsx":
			return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case ".ppt":
			return "application/vnd.ms-powerpoint"
		case ".pptx":
			return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		case ".heic":
			return "image/heic"
		case ".avif":
			return "image/avif"
		}
	}

	return contentType
}

// calculateSHA256 calculates the SHA-256 hash of a file
func calculateSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
