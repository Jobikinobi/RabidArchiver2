package main

import (
	"fmt"
	"os"

	"github.com/jth/archiver/internal/interactive"
	"github.com/spf13/cobra"
)

// newInteractiveCommand creates a command for interactive drive selection and backup
func newInteractiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "Start interactive mode for drive selection and backup",
		Long: `Start the interactive mode to search and select drives for backup.
This mode guides you through the process of selecting external drives,
configuring backup options, and initiating the backup process.`,
		Run: executeInteractive,
	}

	return cmd
}

// executeInteractive runs the interactive mode
func executeInteractive(cmd *cobra.Command, args []string) {
	// Create CLI interactive instance
	cli := interactive.New()

	// Display welcome message
	fmt.Println("Welcome to Archiver Interactive Mode!")
	fmt.Println("=====================================")
	fmt.Println("This mode will guide you through the process of selecting drives and configuring backup options.")

	// Select drives
	selectedDrives, err := cli.SelectDrives()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if cli.HandleAPIKeyErrors(err, appConfig) {
			// If API keys were updated, try again
			selectedDrives, err = cli.SelectDrives()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			os.Exit(1)
		}
	}

	fmt.Printf("\nSelected %d drive(s) for processing.\n", len(selectedDrives))

	// Configure backup options
	backupOptions, err := cli.ConfigureBackupOptions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error configuring backup options: %v\n", err)
		os.Exit(1)
	}

	// Add selected drives to options
	backupOptions.SelectedDrives = selectedDrives

	// Verify API keys are available
	if backupOptions.BackupProvider == "b2" {
		if err := appConfig.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "API key error: %v\n", err)
			if cli.HandleAPIKeyErrors(err, appConfig) {
				// API keys were updated
				fmt.Println("API keys updated successfully.")
			} else {
				fmt.Println("Warning: Continuing with missing API credentials. Upload may fail.")
			}
		}
	}

	// Display summary
	fmt.Println("\nBackup Summary:")
	fmt.Println("---------------")
	fmt.Println("Selected drives:")
	for i, drive := range selectedDrives {
		fmt.Printf("  %d. %s\n", i+1, drive)
	}

	fmt.Printf("\nBackup provider: %s\n", backupOptions.BackupProvider)
	fmt.Printf("Create local copy: %t\n", backupOptions.CreateLocalCopy)
	fmt.Printf("Delete after upload: %t\n", backupOptions.DeleteAfterUpload)
	fmt.Printf("Compress files: %t\n", backupOptions.CompressBeforeUpload)
	fmt.Printf("Encrypt files: %t\n", backupOptions.EncryptBeforeUpload)

	// Confirm before starting
	fmt.Println("\nReady to start the backup process. Continue? (y/n) [y]:")
	fmt.Print("> ")
	cli.Scanner.Scan()
	response := cli.Scanner.Text()
	if response == "n" || response == "no" {
		fmt.Println("Backup cancelled.")
		os.Exit(0)
	}

	// Start backup process
	fmt.Println("\nStarting backup process...")
	executeBackup(backupOptions)
}

// executeBackup handles the actual backup process based on options
func executeBackup(options *interactive.BackupOptions) {
	// For now, just print what we would do
	fmt.Println("Simulating backup process:")

	for _, drive := range options.SelectedDrives {
		fmt.Printf("Processing drive: %s\n", drive)

		// Simulate some processing
		fmt.Println("  - Scanning files...")
		fmt.Println("  - Processing metadata...")

		if options.CompressBeforeUpload {
			fmt.Println("  - Compressing files...")
		}

		if options.EncryptBeforeUpload {
			fmt.Println("  - Encrypting files...")
		}

		fmt.Println("  - Uploading to backup provider...")

		if options.CreateLocalCopy {
			fmt.Println("  - Creating local copy...")
		}

		if options.DeleteAfterUpload {
			fmt.Println("  - Marking files for deletion after upload verification...")
		}

		fmt.Println("  - Drive processing complete!")
	}

	fmt.Println("\nBackup process completed successfully!")
}
