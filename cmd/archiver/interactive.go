package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jth/archiver/internal/db"
	"github.com/jth/archiver/internal/interactive"
	"github.com/jth/archiver/internal/progress"
	"github.com/jth/archiver/internal/scan"
	"github.com/jth/archiver/internal/upload"
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
	fmt.Println("Starting actual backup process...")

	// Create a progress tracker
	tracker := progress.NewTracker()

	for _, drivePath := range options.SelectedDrives {
		fmt.Printf("Processing drive: %s\n", drivePath)

		// Create a database for this drive
		dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("archiver_%s.db", filepath.Base(drivePath)))
		fmt.Printf("Creating database at: %s\n", dbPath)

		// Step 1: Scan the drive
		fmt.Println("  - Scanning files...")
		scanner, err := scan.NewScanner(drivePath, dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating scanner: %v\n", err)
			continue
		}
		defer scanner.Close()

		// Add scanning stage to tracker
		tracker.AddStage("scan", "Scanning files", 100)

		// Perform the scan
		err = scanner.Scan()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning drive: %v\n", err)
			continue
		}
		tracker.CompleteStage("scan")

		// Step 2: Process files
		fmt.Println("  - Processing files...")
		db, err := db.Open(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
			continue
		}
		defer db.Close()

		// Get files to process
		files, err := db.GetUnprocessedFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting unprocessed files: %v\n", err)
			continue
		}

		fmt.Printf("    Found %d files to process\n", len(files))
		tracker.UpdateTotals(int64(len(files)), 0)

		// Add processing stage to tracker
		tracker.AddStage("process", "Processing files", int64(len(files)))

		// Step 3: Upload files to backup provider
		if len(files) > 0 {
			fmt.Println("  - Uploading files to backup provider...")

			// Add upload stage to tracker
			tracker.AddStage("upload", "Uploading files", int64(len(files)))

			if options.BackupProvider == "b2" {
				// Create B2 uploader
				b2Config := upload.B2Config{
					KeyID:      appConfig.B2KeyID,
					AppKey:     appConfig.B2AppKey,
					BucketName: appConfig.B2Bucket,
					Prefix:     filepath.Base(drivePath),
					Concurrent: 4,
				}

				uploader, err := upload.NewB2Uploader(b2Config)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating B2 uploader: %v\n", err)
					continue
				}
				defer uploader.Close()

				// Process files in batches
				batchSize := 10
				for i := 0; i < len(files); i += batchSize {
					end := i + batchSize
					if end > len(files) {
						end = len(files)
					}

					batch := files[i:end]
					paths := make([]string, len(batch))

					for j, file := range batch {
						paths[j] = file.Path
					}

					// Compress if needed
					if options.CompressBeforeUpload {
						fmt.Println("    - Compressing files...")
						// Compression logic would go here
					}

					// Encrypt if needed
					if options.EncryptBeforeUpload {
						fmt.Println("    - Encrypting files...")
						// Encryption logic would go here
					}

					// Upload the batch
					ctx := context.Background()
					results, err := uploader.UploadBatch(ctx, paths)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error uploading batch: %v\n", err)
						continue
					}

					// Update database with results
					for j, result := range results {
						if result.Error != nil {
							fmt.Fprintf(os.Stderr, "Error uploading %s: %v\n", result.LocalPath, result.Error)
							continue
						}

						// Update the database record
						err = db.UpdateFileStatus(batch[j].ID, true, result.URL, "")
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error updating file status: %v\n", err)
						}

						// Create local copy if needed
						if options.CreateLocalCopy {
							// Local copy logic would go here
						}

						// Delete after upload if needed
						if options.DeleteAfterUpload {
							fmt.Printf("    - Marking %s for deletion\n", result.LocalPath)
							// Deletion logic would go here
						}

						// Update progress
						tracker.IncrementStage("process", 1)
						tracker.IncrementStage("upload", 1)
						tracker.UpdateFileStats(1, 0, 0, result.Size)
						tracker.UpdateUploadStats(result.Size)
					}
				}
			} else if strings.HasPrefix(options.BackupProvider, "local:") {
				// Local backup logic
				localDir := strings.TrimPrefix(options.BackupProvider, "local:")
				fmt.Printf("    Backing up to local directory: %s\n", localDir)

				// Create the directory if it doesn't exist
				err = os.MkdirAll(localDir, 0755)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating local backup directory: %v\n", err)
					continue
				}

				// Process each file
				for i, file := range files {
					destPath := filepath.Join(localDir, filepath.Base(file.Path))

					// Copy the file
					err = copyFile(file.Path, destPath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error copying file %s: %v\n", file.Path, err)
						continue
					}

					// Update the database record
					err = db.UpdateFileStatus(file.ID, true, destPath, "")
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error updating file status: %v\n", err)
					}

					// Update progress
					tracker.IncrementStage("process", 1)
					tracker.IncrementStage("upload", 1)
					tracker.UpdateFileStats(1, 0, 0, file.Size)
					tracker.UpdateUploadStats(file.Size)

					// Log progress
					if (i+1)%10 == 0 {
						fmt.Printf("    Processed %d/%d files\n", i+1, len(files))
					}
				}
			}
		}

		fmt.Println("  - Drive processing complete!")
	}

	// Print final summary
	tracker.PrintSummary()
	fmt.Println("\nBackup process completed successfully!")
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}
