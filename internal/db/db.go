package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FileStatus represents the processing status of a file
type FileStatus struct {
	ID           int64
	Path         string
	RelativePath string
	Size         int64
	ModTime      time.Time
	IsDir        bool
	ContentType  string
	SHA256       string
	Processed    bool
	UploadedURL  string
	UploadTime   sql.NullTime
	Summary      string
}

// DB provides a database connection and utility functions
type DB struct {
	conn *sql.DB
}

// Open opens a connection to the database
func Open(dbPath string) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	// Verify connection
	if err := db.conn.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetFileByPath retrieves a file by its path
func (db *DB) GetFileByPath(path string) (*FileStatus, error) {
	query := `
	SELECT id, path, relative_path, size, mod_time, is_dir, content_type, 
	       sha256, processed, uploaded_url, upload_time, summary
	FROM files
	WHERE path = ?
	`

	row := db.conn.QueryRow(query, path)

	var file FileStatus
	err := row.Scan(
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

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &file, nil
}

// GetUnprocessedFiles retrieves all unprocessed files
func (db *DB) GetUnprocessedFiles() ([]*FileStatus, error) {
	query := `
	SELECT id, path, relative_path, size, mod_time, is_dir, content_type, 
	       sha256, processed, uploaded_url, upload_time, summary
	FROM files
	WHERE processed = FALSE AND is_dir = FALSE
	ORDER BY path
	`

	rows, err := db.conn.Query(query)
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

// GetFilesByType retrieves files by MIME type prefix
func (db *DB) GetFilesByType(typePrefix string) ([]*FileStatus, error) {
	query := `
	SELECT id, path, relative_path, size, mod_time, is_dir, content_type, 
	       sha256, processed, uploaded_url, upload_time, summary
	FROM files
	WHERE content_type LIKE ? AND is_dir = FALSE
	ORDER BY path
	`

	rows, err := db.conn.Query(query, typePrefix+"%")
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

// UpdateFileStatus updates the status of a file
func (db *DB) UpdateFileStatus(id int64, processed bool, uploadedURL string, summary string) error {
	query := `
	UPDATE files
	SET processed = ?, uploaded_url = ?, upload_time = ?, summary = ?
	WHERE id = ?
	`

	_, err := db.conn.Exec(query, processed, uploadedURL, time.Now(), summary, id)
	return err
}

// GetStats returns statistics about the files in the database
func (db *DB) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Total files
	var totalFiles int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM files WHERE is_dir = FALSE").Scan(&totalFiles)
	if err != nil {
		return nil, err
	}
	stats["totalFiles"] = totalFiles

	// Total directories
	var totalDirs int64
	err = db.conn.QueryRow("SELECT COUNT(*) FROM files WHERE is_dir = TRUE").Scan(&totalDirs)
	if err != nil {
		return nil, err
	}
	stats["totalDirs"] = totalDirs

	// Processed files
	var processedFiles int64
	err = db.conn.QueryRow("SELECT COUNT(*) FROM files WHERE processed = TRUE AND is_dir = FALSE").Scan(&processedFiles)
	if err != nil {
		return nil, err
	}
	stats["processedFiles"] = processedFiles

	// Total size
	var totalSize int64
	err = db.conn.QueryRow("SELECT SUM(size) FROM files WHERE is_dir = FALSE").Scan(&totalSize)
	if err != nil {
		return nil, err
	}
	stats["totalSize"] = totalSize

	return stats, nil
}
