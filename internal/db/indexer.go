package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

// IndexConfig represents the configuration for the full-text search index
type IndexConfig struct {
	// Path to the directory where the index will be stored
	IndexDir string
	// Whether to index file content summaries
	IndexSummaries bool
}

// SearchResult represents a search result item
type SearchResult struct {
	ID       string
	Path     string
	Score    float64
	Snippet  string
	IsDir    bool
	Size     int64
	ModTime  time.Time
	Metadata map[string]interface{}
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query     string
	Limit     int
	Offset    int
	SortBy    string
	SortDesc  bool
	FieldName string // Restrict search to a specific field
}

// FileIndex represents the indexed file document
type FileIndex struct {
	ID           string
	Path         string
	RelativePath string
	Name         string
	Extension    string
	Size         int64
	ModTime      time.Time
	IsDir        bool
	ContentType  string
	Summary      string
	UploadedURL  string
	UpdatedAt    time.Time
}

// BleveIndexer provides full-text search capabilities
type BleveIndexer struct {
	config IndexConfig
	index  bleve.Index
	db     *DB
}

// NewIndexer creates a new full-text search indexer
func NewIndexer(config IndexConfig, db *DB) (*BleveIndexer, error) {
	// Create index directory if it doesn't exist
	if err := os.MkdirAll(config.IndexDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	indexPath := filepath.Join(config.IndexDir, "fileindex.bleve")

	var index bleve.Index
	var err error

	// Open existing index or create a new one
	if _, err = os.Stat(indexPath); os.IsNotExist(err) {
		// Create a new index
		indexMapping := createIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	} else {
		// Open existing index
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open index: %w", err)
		}
	}

	return &BleveIndexer{
		config: config,
		index:  index,
		db:     db,
	}, nil
}

// createIndexMapping creates a Bleve index mapping for file documents
func createIndexMapping() mapping.IndexMapping {
	// Create a mapping for file documents
	indexMapping := bleve.NewIndexMapping()

	// Document mapping for FileIndex
	documentMapping := bleve.NewDocumentMapping()

	// Text fields with full-text indexing
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = true
	textFieldMapping.IncludeInAll = true
	textFieldMapping.IncludeTermVectors = true
	textFieldMapping.Analyzer = "standard"

	documentMapping.AddFieldMappingsAt("Path", textFieldMapping)
	documentMapping.AddFieldMappingsAt("RelativePath", textFieldMapping)
	documentMapping.AddFieldMappingsAt("Name", textFieldMapping)
	documentMapping.AddFieldMappingsAt("Summary", textFieldMapping)

	// Keyword fields
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Store = true
	keywordFieldMapping.IncludeInAll = true
	keywordFieldMapping.Analyzer = "keyword"

	documentMapping.AddFieldMappingsAt("Extension", keywordFieldMapping)
	documentMapping.AddFieldMappingsAt("ContentType", keywordFieldMapping)

	// Numeric fields
	numericFieldMapping := bleve.NewNumericFieldMapping()
	numericFieldMapping.Store = true

	documentMapping.AddFieldMappingsAt("Size", numericFieldMapping)

	// Date fields
	dateTimeFieldMapping := bleve.NewDateTimeFieldMapping()
	dateTimeFieldMapping.Store = true

	documentMapping.AddFieldMappingsAt("ModTime", dateTimeFieldMapping)
	documentMapping.AddFieldMappingsAt("UpdatedAt", dateTimeFieldMapping)

	// Boolean fields
	booleanFieldMapping := bleve.NewBooleanFieldMapping()
	booleanFieldMapping.Store = true

	documentMapping.AddFieldMappingsAt("IsDir", booleanFieldMapping)

	// Add the document mapping to the index
	indexMapping.AddDocumentMapping("fileindex", documentMapping)

	return indexMapping
}

// Close closes the index
func (idx *BleveIndexer) Close() error {
	return idx.index.Close()
}

// IndexFile indexes a single file
func (idx *BleveIndexer) IndexFile(file *FileStatus) error {
	if file == nil {
		return fmt.Errorf("cannot index nil file")
	}

	// Extract file name and extension
	name := filepath.Base(file.Path)
	extension := strings.ToLower(filepath.Ext(file.Path))

	// Create a document to index
	doc := FileIndex{
		ID:           fmt.Sprintf("%d", file.ID),
		Path:         file.Path,
		RelativePath: file.RelativePath,
		Name:         name,
		Extension:    extension,
		Size:         file.Size,
		ModTime:      file.ModTime,
		IsDir:        file.IsDir,
		ContentType:  file.ContentType,
		UploadedURL:  file.UploadedURL,
		UpdatedAt:    time.Now(),
	}

	// Include summary if configured and available
	if idx.config.IndexSummaries && file.Summary != "" {
		doc.Summary = file.Summary
	}

	// Index the document
	return idx.index.Index(doc.ID, doc)
}

// RemoveFile removes a file from the index
func (idx *BleveIndexer) RemoveFile(fileID int64) error {
	id := fmt.Sprintf("%d", fileID)
	return idx.index.Delete(id)
}

// UpdateFile updates a file in the index
func (idx *BleveIndexer) UpdateFile(file *FileStatus) error {
	// First remove the existing document
	if err := idx.RemoveFile(file.ID); err != nil {
		// Ignore "document not found" errors
		if !strings.Contains(err.Error(), "document not found") {
			return err
		}
	}

	// Then add the updated document
	return idx.IndexFile(file)
}

// BuildIndex builds or rebuilds the full index from the database
func (idx *BleveIndexer) BuildIndex() (int, error) {
	// Get all files from the database
	query := `
	SELECT id, path, relative_path, size, mod_time, is_dir, content_type, 
	       sha256, processed, uploaded_url, upload_time, summary
	FROM files
	ORDER BY id
	`

	rows, err := idx.db.conn.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// Create a batch for more efficient indexing
	batch := idx.index.NewBatch()
	count := 0
	batchSize := 100

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
			return count, err
		}

		// Extract file name and extension
		name := filepath.Base(file.Path)
		extension := strings.ToLower(filepath.Ext(file.Path))

		// Create a document to index
		doc := FileIndex{
			ID:           fmt.Sprintf("%d", file.ID),
			Path:         file.Path,
			RelativePath: file.RelativePath,
			Name:         name,
			Extension:    extension,
			Size:         file.Size,
			ModTime:      file.ModTime,
			IsDir:        file.IsDir,
			ContentType:  file.ContentType,
			UploadedURL:  file.UploadedURL,
			UpdatedAt:    time.Now(),
		}

		// Include summary if configured and available
		if idx.config.IndexSummaries && file.Summary != "" {
			doc.Summary = file.Summary
		}

		// Add to batch
		if err := batch.Index(doc.ID, doc); err != nil {
			return count, err
		}
		count++

		// Execute batch if it reaches the batch size
		if count%batchSize == 0 {
			if err := idx.index.Batch(batch); err != nil {
				return count, err
			}
			batch = idx.index.NewBatch()
		}
	}

	// Execute the final batch if there are any documents left
	if batch.Size() > 0 {
		if err := idx.index.Batch(batch); err != nil {
			return count, err
		}
	}

	if err := rows.Err(); err != nil {
		return count, err
	}

	return count, nil
}

// Search performs a search on the index
func (idx *BleveIndexer) Search(request SearchRequest) ([]SearchResult, error) {
	// Set defaults if not specified
	if request.Limit <= 0 {
		request.Limit = 10
	}

	// Create a query based on the search request
	var searchQuery query.Query

	if request.Query == "" {
		// If no query is provided, match all documents
		searchQuery = bleve.NewMatchAllQuery()
	} else if request.FieldName != "" {
		// Search in a specific field
		matchQuery := bleve.NewMatchQuery(request.Query)
		matchQuery.SetField(request.FieldName)
		searchQuery = matchQuery
	} else {
		// Search in all fields
		searchQuery = bleve.NewQueryStringQuery(request.Query)
	}

	// Create the search request
	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Size = request.Limit
	searchRequest.From = request.Offset
	searchRequest.Fields = []string{"*"}
	searchRequest.IncludeLocations = true

	// Set up sorting
	if request.SortBy != "" {
		sortOrder := "asc"
		if request.SortDesc {
			sortOrder = "desc"
		}
		searchRequest.SortBy([]string{request.SortBy + ":" + sortOrder})
	}

	// Set up highlighting for snippets
	searchRequest.Highlight = bleve.NewHighlight()

	// Execute the search
	searchResults, err := idx.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// Process the results
	var results []SearchResult
	for _, hit := range searchResults.Hits {
		// Extract fields from the document
		path, _ := hit.Fields["Path"].(string)
		size, _ := hit.Fields["Size"].(float64)
		isDir, _ := hit.Fields["IsDir"].(bool)

		// Extract modTime
		var modTime time.Time
		if modTimeStr, ok := hit.Fields["ModTime"].(string); ok {
			if t, err := time.Parse(time.RFC3339, modTimeStr); err == nil {
				modTime = t
			}
		}

		// Extract snippet from highlighted fragments
		snippet := ""
		if fragments, ok := hit.Fragments["Summary"]; ok && len(fragments) > 0 {
			snippet = fragments[0]
		} else if fragments, ok := hit.Fragments["Path"]; ok && len(fragments) > 0 {
			snippet = fragments[0]
		}

		// Create a search result
		result := SearchResult{
			ID:       hit.ID,
			Path:     path,
			Score:    hit.Score,
			Snippet:  snippet,
			IsDir:    isDir,
			Size:     int64(size),
			ModTime:  modTime,
			Metadata: hit.Fields,
		}

		results = append(results, result)
	}

	return results, nil
}

// GetStats returns statistics about the index
func (idx *BleveIndexer) GetStats() (map[string]interface{}, error) {
	stats := idx.index.Stats()
	if stats == nil {
		return nil, fmt.Errorf("failed to get index statistics")
	}

	// Extract available stats into a map
	statsMap := make(map[string]interface{})

	// Get document count, handling both return values
	docCount, err := idx.index.DocCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}
	statsMap["documentCount"] = docCount

	// Add the raw stats object
	statsMap["indexStats"] = stats

	return statsMap, nil
}

// GetDocumentCount returns the number of documents in the index
func (idx *BleveIndexer) GetDocumentCount() (uint64, error) {
	return idx.index.DocCount()
}
