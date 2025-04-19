package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIndexer(t *testing.T) {
	// Create temporary directory for the test
	tempDir, err := os.MkdirTemp("", "indexer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create temporary database file
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create temporary index directory
	indexDir := filepath.Join(tempDir, "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatalf("Failed to create index directory: %v", err)
	}

	// Create a test file in the database
	testFile := &FileStatus{
		ID:           1,
		Path:         "/test/path/file.txt",
		RelativePath: "path/file.txt",
		Size:         1024,
		ModTime:      time.Now(),
		IsDir:        false,
		ContentType:  "text/plain",
		SHA256:       "abcdef1234567890",
		Processed:    true,
		UploadedURL:  "https://example.com/file.txt",
		Summary:      "This is a test file",
	}

	// Insert the test file into the database
	err = insertTestFile(db, testFile)
	if err != nil {
		t.Fatalf("Failed to insert test file into database: %v", err)
	}

	// Create the indexer
	config := IndexConfig{
		IndexDir:       indexDir,
		IndexSummaries: true,
	}
	indexer, err := NewIndexer(config, db)
	if err != nil {
		t.Fatalf("Failed to create indexer: %v", err)
	}
	defer indexer.Close()

	// Test indexing a file
	t.Run("IndexFile", func(t *testing.T) {
		err := indexer.IndexFile(testFile)
		if err != nil {
			t.Fatalf("Failed to index file: %v", err)
		}

		// Verify file was indexed
		count, err := indexer.GetDocumentCount()
		if err != nil {
			t.Fatalf("Failed to get document count: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 document in index, got %d", count)
		}
	})

	// Test searching for the file
	t.Run("Search", func(t *testing.T) {
		request := SearchRequest{
			Query:  "test",
			Limit:  10,
			Offset: 0,
		}
		results, err := indexer.Search(request)
		if err != nil {
			t.Fatalf("Failed to search index: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 search result, got %d", len(results))
		}

		result := results[0]
		if result.Path != testFile.Path {
			t.Errorf("Expected Path to be %s, got %s", testFile.Path, result.Path)
		}
		if result.IsDir != testFile.IsDir {
			t.Errorf("Expected IsDir to be %v, got %v", testFile.IsDir, result.IsDir)
		}
	})

	// Test field-specific search
	t.Run("FieldSearch", func(t *testing.T) {
		request := SearchRequest{
			Query:     "test",
			FieldName: "Summary",
			Limit:     10,
		}
		results, err := indexer.Search(request)
		if err != nil {
			t.Fatalf("Failed to search specific field: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 search result, got %d", len(results))
		}
	})

	// Test removing a file from the index
	t.Run("RemoveFile", func(t *testing.T) {
		err := indexer.RemoveFile(testFile.ID)
		if err != nil {
			t.Fatalf("Failed to remove file from index: %v", err)
		}

		// Verify file was removed
		count, err := indexer.GetDocumentCount()
		if err != nil {
			t.Fatalf("Failed to get document count: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 documents in index after removal, got %d", count)
		}

		// Search should return no results
		request := SearchRequest{
			Query: "test",
			Limit: 10,
		}
		results, err := indexer.Search(request)
		if err != nil {
			t.Fatalf("Failed to search index: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 search results after removal, got %d", len(results))
		}
	})

	// Test rebuilding the index
	t.Run("BuildIndex", func(t *testing.T) {
		// Add another test file
		testFile2 := &FileStatus{
			ID:           2,
			Path:         "/test/path/file2.doc",
			RelativePath: "path/file2.doc",
			Size:         2048,
			ModTime:      time.Now(),
			IsDir:        false,
			ContentType:  "application/msword",
			SHA256:       "09876543210abcdef",
			Processed:    true,
			UploadedURL:  "https://example.com/file2.doc",
			Summary:      "This is another test file with some content",
		}
		err := insertTestFile(db, testFile2)
		if err != nil {
			t.Fatalf("Failed to insert second test file: %v", err)
		}

		// Build the index
		count, err := indexer.BuildIndex()
		if err != nil {
			t.Fatalf("Failed to build index: %v", err)
		}
		if count != 2 {
			t.Errorf("BuildIndex returned count %d, expected 2", count)
		}

		// Verify document count
		docCount, err := indexer.GetDocumentCount()
		if err != nil {
			t.Fatalf("Failed to get document count: %v", err)
		}
		if docCount != 2 {
			t.Errorf("Expected 2 documents in index after rebuild, got %d", docCount)
		}
	})

	// Test getting stats
	t.Run("GetStats", func(t *testing.T) {
		stats, err := indexer.GetStats()
		if err != nil {
			t.Fatalf("Failed to get index stats: %v", err)
		}
		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}
		if _, ok := stats["documentCount"]; !ok {
			t.Errorf("Expected documentCount in stats, got %v", stats)
		}
	})
}

// Helper function to insert a test file into the database
func insertTestFile(db *DB, file *FileStatus) error {
	query := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY,
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
	);`
	_, err := db.conn.Exec(query)
	if err != nil {
		return err
	}

	query = `
	INSERT INTO files 
	(id, path, relative_path, size, mod_time, is_dir, content_type, 
	 sha256, processed, uploaded_url, upload_time, summary)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	uploadTime := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	_, err = db.conn.Exec(
		query,
		file.ID,
		file.Path,
		file.RelativePath,
		file.Size,
		file.ModTime,
		file.IsDir,
		file.ContentType,
		file.SHA256,
		file.Processed,
		file.UploadedURL,
		uploadTime,
		file.Summary,
	)
	return err
}
