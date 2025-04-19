package main

import (
	"fmt"
	"os"

	"github.com/jth/archiver/internal/config"
	"github.com/spf13/cobra"
)

var (
	configPath      string
	sourcePath      string
	b2KeyID         string
	b2AppKey        string
	bucket          string
	summarize       string
	stubMode        string
	costCap         float64
	appConfig       *config.Config
	debugMode       bool
	interactiveMode bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "archiver",
		Short: "Archiver - Process, summarize, and backup files to B2",
		Long: `Archiver is a CLI tool that ingests an external drive, transcodes videos,
summarizes documents, uploads to Backblaze B2, and provides a searchable index.`,
		PersistentPreRun: loadConfig,
		Run:              executeArchiver,
	}

	// Define flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "./config.json", "Path to config file (optional)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug output")
	rootCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Path to the source directory (required)")
	rootCmd.Flags().StringVar(&b2KeyID, "b2-key-id", "", "Backblaze B2 Key ID (required)")
	rootCmd.Flags().StringVar(&b2AppKey, "b2-app-key", "", "Backblaze B2 Application Key (required)")
	rootCmd.Flags().StringVar(&bucket, "bucket", "", "Backblaze B2 bucket name (required)")
	rootCmd.Flags().StringVar(&summarize, "summarize", "default", "Summarization level: none, basic, default, or full")
	rootCmd.Flags().StringVar(&stubMode, "stub-mode", "webloc", "Local stub format: webloc, shortcut, or none")
	rootCmd.Flags().Float64Var(&costCap, "cost-cap", 5.0, "Maximum LLM spend in USD")
	rootCmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Start in interactive mode")

	// Only mark flags as required if not in interactive mode
	isInteractiveArg := false
	if len(os.Args) > 1 {
		isInteractiveArg = os.Args[1] == "interactive" || os.Args[1] == "help"
	}

	if !interactiveMode && !isInteractiveArg {
		rootCmd.MarkFlagRequired("source")
		rootCmd.MarkFlagRequired("b2-key-id")
		rootCmd.MarkFlagRequired("b2-app-key")
		rootCmd.MarkFlagRequired("bucket")
	}

	// Add subcommands
	rootCmd.AddCommand(newSearchCommand())
	rootCmd.AddCommand(newInteractiveCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadConfig(cmd *cobra.Command, args []string) {
	// First, try to load from config file if it exists
	if _, statErr := os.Stat(configPath); statErr == nil {
		var err error
		appConfig, err = config.LoadFromFile(configPath)
		if err != nil {
			fmt.Printf("Warning: Could not load config file: %v\n", err)
			// Continue to load from env and flags
		} else if debugMode {
			fmt.Printf("Loaded configuration from: %s\n", configPath)
		}
	}

	// If no config loaded, use environment variables
	if appConfig == nil {
		appConfig = config.LoadFromEnv()
		if debugMode {
			fmt.Println("Loaded configuration from environment variables")
		}
	}

	// Override with command line flags if provided
	if cmd.Flags().Changed("b2-key-id") {
		appConfig.B2KeyID = b2KeyID
	} else if appConfig.B2KeyID != "" {
		b2KeyID = appConfig.B2KeyID
	}

	if cmd.Flags().Changed("b2-app-key") {
		appConfig.B2AppKey = b2AppKey
	} else if appConfig.B2AppKey != "" {
		b2AppKey = appConfig.B2AppKey
	}

	if cmd.Flags().Changed("bucket") {
		appConfig.B2Bucket = bucket
	} else if appConfig.B2Bucket != "" {
		bucket = appConfig.B2Bucket
	}

	if cmd.Flags().Changed("summarize") {
		appConfig.Summarize = summarize
	} else if appConfig.Summarize != "" {
		summarize = appConfig.Summarize
	}

	if cmd.Flags().Changed("stub-mode") {
		appConfig.StubMode = stubMode
	} else if appConfig.StubMode != "" {
		stubMode = appConfig.StubMode
	}

	if cmd.Flags().Changed("cost-cap") {
		appConfig.CostCapUSD = costCap
	} else if appConfig.CostCapUSD != 0 {
		costCap = appConfig.CostCapUSD
	}

	// If interactive flag is used, start the interactive command
	if interactiveMode && cmd.Name() != "interactive" && cmd.Name() != "help" {
		// We're in root command with interactive flag - pass control to interactive command
		interactiveCmd := newInteractiveCommand()
		interactiveCmd.Run(cmd, args)
		os.Exit(0)
	}

	// Print API key info in debug mode
	if debugMode {
		fmt.Println("Configuration loaded successfully")
		fmt.Printf("B2 Key ID: %s...\n", maskString(appConfig.B2KeyID))
		fmt.Printf("Anthropic API Key: %s...\n", maskString(appConfig.AnthropicAPIKey))
		fmt.Printf("OpenAI API Key: %s...\n", maskString(appConfig.OpenAIAPIKey))
		fmt.Printf("Mistral API Key: %s...\n", maskString(appConfig.MistralAPIKey))
		fmt.Printf("Grok API Key: %s...\n", maskString(appConfig.GrokAPIKey))
	}
}

// maskString returns a masked version of a string, showing only the first 4 characters
func maskString(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func executeArchiver(cmd *cobra.Command, args []string) {
	// If interactive flag is set, this would have already been handled in loadConfig
	// This is just a fallback
	if interactiveMode {
		interactiveCmd := newInteractiveCommand()
		interactiveCmd.Run(cmd, args)
		return
	}

	fmt.Println("Starting Archiver...")
	fmt.Printf("Processing source: %s\n", sourcePath)
	fmt.Printf("Using B2 bucket: %s\n", bucket)
	fmt.Printf("Summarization level: %s\n", summarize)
	fmt.Printf("Stub mode: %s\n", stubMode)
	fmt.Printf("Cost cap: $%.2f USD\n", costCap)

	// Main processing pipeline will be implemented here
	fmt.Println("Archiver completed successfully.")
}
