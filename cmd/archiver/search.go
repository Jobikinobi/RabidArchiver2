package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jth/archiver/internal/db"
	"github.com/spf13/cobra"
)

var (
	indexDir     string
	query        string
	fieldName    string
	limit        int
	offset       int
	sortBy       string
	sortDesc     bool
	dbFilePath   string
	outputFormat string
)

// searchCmd represents the search command
func newSearchCommand() *cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search for files in the archive",
		Long: `Search for files in the archive using full-text search.
Examples:
  archiver search --query "document about finance"
  archiver search --query "image" --field "ContentType" --limit 20
  archiver search --query "report" --sort-by "ModTime" --sort-desc`,
		Run: executeSearch,
	}

	// Add flags
	searchCmd.Flags().StringVar(&indexDir, "index-dir", "./index", "Directory containing the search index")
	searchCmd.Flags().StringVar(&dbFilePath, "db", "./archive.db", "Path to the archive database")
	searchCmd.Flags().StringVarP(&query, "query", "q", "", "Search query (required)")
	searchCmd.Flags().StringVarP(&fieldName, "field", "f", "", "Restrict search to this field (e.g., Path, Name, Summary)")
	searchCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of results to return")
	searchCmd.Flags().IntVarP(&offset, "offset", "o", 0, "Number of results to skip (for pagination)")
	searchCmd.Flags().StringVar(&sortBy, "sort-by", "", "Field to sort by (e.g., ModTime, Size, Path)")
	searchCmd.Flags().BoolVar(&sortDesc, "sort-desc", false, "Sort in descending order")
	searchCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format: text, json")

	// Mark required flags
	searchCmd.MarkFlagRequired("query")

	return searchCmd
}

// executeSearch performs the search operation
func executeSearch(cmd *cobra.Command, args []string) {
	// Create a database connection
	database, err := db.Open(dbFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create index config
	config := db.IndexConfig{
		IndexDir:       indexDir,
		IndexSummaries: true,
	}

	// Create the indexer
	indexer, err := db.NewIndexer(config, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating indexer: %v\n", err)
		os.Exit(1)
	}
	defer indexer.Close()

	// Create the search request
	request := db.SearchRequest{
		Query:     query,
		FieldName: fieldName,
		Limit:     limit,
		Offset:    offset,
		SortBy:    sortBy,
		SortDesc:  sortDesc,
	}

	// Perform the search
	results, err := indexer.Search(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
		os.Exit(1)
	}

	// Output the results
	if outputFormat == "json" {
		outputJSON(results)
	} else {
		outputText(results, query)
	}

	// Print summary
	fmt.Printf("\nFound %d results for query: %s\n", len(results), query)
}

// outputText prints search results in text format
func outputText(results []db.SearchResult, searchQuery string) {
	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	for i, result := range results {
		// Format the path for display
		displayPath := result.Path
		if home, err := os.UserHomeDir(); err == nil {
			if strings.HasPrefix(displayPath, home) {
				displayPath = "~" + displayPath[len(home):]
			}
		}

		// Format size
		size := formatSize(result.Size)

		// Format time
		timeStr := result.ModTime.Format("Jan 02, 2006")

		// Format type indicator
		typeIndicator := " "
		if result.IsDir {
			typeIndicator = "D"
		}

		// Print result header
		fmt.Printf("\n%d. [%s] %s (%.2f)\n", i+1, typeIndicator, displayPath, result.Score)
		fmt.Printf("   Size: %s | Modified: %s\n", size, timeStr)

		// Print snippet if available
		if result.Snippet != "" {
			fmt.Printf("   %s\n", result.Snippet)
		}

		// Print metadata if available and relevant
		if result.Metadata != nil {
			if contentType, ok := result.Metadata["ContentType"].(string); ok && contentType != "" {
				fmt.Printf("   Type: %s\n", contentType)
			}
		}

		// Add separator after each result
		if i < len(results)-1 {
			fmt.Println("   -----------------------------")
		}
	}
}

// outputJSON prints search results in JSON format
func outputJSON(results []db.SearchResult) {
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// formatSize formats file size in human-readable format
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
