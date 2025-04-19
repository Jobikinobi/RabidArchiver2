package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	sourcePath string
	b2KeyID    string
	b2AppKey   string
	bucket     string
	summarise  string
	stubMode   string
	costCap    float64
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "archiver",
		Short: "Archiver - Process, summarize, and backup files to B2",
		Long: `Archiver is a CLI tool that ingests an external drive, transcodes videos,
summarizes documents, uploads to Backblaze B2, and provides a searchable index.`,
		Run: executeArchiver,
	}

	// Define flags
	rootCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Path to the source directory (required)")
	rootCmd.Flags().StringVar(&b2KeyID, "b2-key-id", "", "Backblaze B2 Key ID (required)")
	rootCmd.Flags().StringVar(&b2AppKey, "b2-app-key", "", "Backblaze B2 Application Key (required)")
	rootCmd.Flags().StringVar(&bucket, "bucket", "", "Backblaze B2 bucket name (required)")
	rootCmd.Flags().StringVar(&summarise, "summarise", "default", "Summarization level: none, basic, default, or full")
	rootCmd.Flags().StringVar(&stubMode, "stub-mode", "webloc", "Local stub format: webloc, shortcut, or none")
	rootCmd.Flags().Float64Var(&costCap, "cost-cap", 5.0, "Maximum LLM spend in USD")

	// Mark required flags
	rootCmd.MarkFlagRequired("source")
	rootCmd.MarkFlagRequired("b2-key-id")
	rootCmd.MarkFlagRequired("b2-app-key")
	rootCmd.MarkFlagRequired("bucket")

	// Environment variable overrides
	if envKeyID := os.Getenv("B2_KEY_ID"); envKeyID != "" && b2KeyID == "" {
		b2KeyID = envKeyID
	}
	if envAppKey := os.Getenv("B2_APP_KEY"); envAppKey != "" && b2AppKey == "" {
		b2AppKey = envAppKey
	}
	if envCostCap := os.Getenv("COST_CAP_USD"); envCostCap != "" {
		fmt.Sscanf(envCostCap, "%f", &costCap)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func executeArchiver(cmd *cobra.Command, args []string) {
	fmt.Println("Starting Archiver...")
	fmt.Printf("Processing source: %s\n", sourcePath)
	fmt.Printf("Using B2 bucket: %s\n", bucket)
	fmt.Printf("Summarization level: %s\n", summarise)
	fmt.Printf("Stub mode: %s\n", stubMode)
	fmt.Printf("Cost cap: $%.2f USD\n", costCap)

	// Main processing pipeline will be implemented here
	fmt.Println("Archiver completed successfully.")
}
