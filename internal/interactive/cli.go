package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jth/archiver/internal/config"
	"github.com/jth/archiver/internal/drives"
)

// CLI handles interactive command-line operations
type CLI struct {
	Scanner *bufio.Scanner
}

// BackupOptions represents the backup configuration options
type BackupOptions struct {
	CreateLocalCopy      bool
	DeleteAfterUpload    bool
	CompressBeforeUpload bool
	EncryptBeforeUpload  bool
	BackupProvider       string
	SelectedDrives       []string
}

// New creates a new CLI instance
func New() *CLI {
	return &CLI{
		Scanner: bufio.NewScanner(os.Stdin),
	}
}

// SelectDrives displays available drives and lets the user select which to process
func (c *CLI) SelectDrives() ([]string, error) {
	fmt.Println("Scanning for external drives...")

	drives, err := drives.ListDrives()
	if err != nil {
		return nil, fmt.Errorf("error listing drives: %w", err)
	}

	if len(drives) == 0 {
		return nil, fmt.Errorf("no external drives found")
	}

	fmt.Println("\nAvailable drives:")
	for i, drive := range drives {
		fmt.Printf("%d. %s (%s) - %s free of %s\n",
			i+1, drive.Name, drive.MountPoint, drive.FreeSpace, drive.Size)
	}

	fmt.Println("\nSelect drives to process (comma separated numbers, or 'all'):")
	fmt.Print("> ")

	c.Scanner.Scan()
	input := strings.TrimSpace(c.Scanner.Text())

	if strings.ToLower(input) == "all" {
		var selectedDrives []string
		for _, drive := range drives {
			selectedDrives = append(selectedDrives, drive.MountPoint)
		}
		return selectedDrives, nil
	}

	// Parse individual selections
	selections := strings.Split(input, ",")
	var selectedDrives []string

	for _, sel := range selections {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}

		num, err := strconv.Atoi(sel)
		if err != nil || num < 1 || num > len(drives) {
			return nil, fmt.Errorf("invalid selection: %s", sel)
		}

		// Convert 1-based user selection to 0-based index
		selectedDrives = append(selectedDrives, drives[num-1].MountPoint)
	}

	if len(selectedDrives) == 0 {
		return nil, fmt.Errorf("no drives selected")
	}

	return selectedDrives, nil
}

// ConfigureBackupOptions allows the user to configure backup options
func (c *CLI) ConfigureBackupOptions() (*BackupOptions, error) {
	options := &BackupOptions{
		CreateLocalCopy:      false,
		DeleteAfterUpload:    false,
		CompressBeforeUpload: true,
		EncryptBeforeUpload:  false,
		BackupProvider:       "b2",
	}

	fmt.Println("\nConfigure backup options:")

	// Create local copy?
	fmt.Println("Create local copy of files before upload? (y/n) [n]:")
	fmt.Print("> ")
	c.Scanner.Scan()
	input := strings.ToLower(strings.TrimSpace(c.Scanner.Text()))
	options.CreateLocalCopy = input == "y" || input == "yes"

	// Delete after upload?
	fmt.Println("Delete files after successful upload? (y/n) [n]:")
	fmt.Print("> ")
	c.Scanner.Scan()
	input = strings.ToLower(strings.TrimSpace(c.Scanner.Text()))
	options.DeleteAfterUpload = input == "y" || input == "yes"

	// Compress before upload?
	fmt.Println("Compress files before upload? (y/n) [y]:")
	fmt.Print("> ")
	c.Scanner.Scan()
	input = strings.ToLower(strings.TrimSpace(c.Scanner.Text()))
	options.CompressBeforeUpload = input != "n" && input != "no"

	// Encrypt before upload?
	fmt.Println("Encrypt files before upload? (y/n) [n]:")
	fmt.Print("> ")
	c.Scanner.Scan()
	input = strings.ToLower(strings.TrimSpace(c.Scanner.Text()))
	options.EncryptBeforeUpload = input == "y" || input == "yes"

	// Select backup provider
	fmt.Println("Select backup provider:")
	fmt.Println("1. Backblaze B2 (default)")
	fmt.Println("2. Local directory")
	fmt.Print("> ")
	c.Scanner.Scan()
	input = strings.TrimSpace(c.Scanner.Text())

	switch input {
	case "1", "":
		options.BackupProvider = "b2"
	case "2":
		options.BackupProvider = "local"
		fmt.Println("Enter local backup directory:")
		fmt.Print("> ")
		c.Scanner.Scan()
		options.BackupProvider = "local:" + strings.TrimSpace(c.Scanner.Text())
	default:
		return nil, fmt.Errorf("invalid provider selection: %s", input)
	}

	return options, nil
}

// PromptAPIKeys prompts the user to enter API keys interactively
func (c *CLI) PromptAPIKeys(cfg *config.Config) error {
	if cfg.B2KeyID == "" {
		fmt.Println("Backblaze B2 Key ID not found in configuration.")
		fmt.Println("Enter Backblaze B2 Key ID:")
		fmt.Print("> ")
		c.Scanner.Scan()
		cfg.B2KeyID = strings.TrimSpace(c.Scanner.Text())
	}

	if cfg.B2AppKey == "" {
		fmt.Println("Backblaze B2 Application Key not found in configuration.")
		fmt.Println("Enter Backblaze B2 Application Key:")
		fmt.Print("> ")
		c.Scanner.Scan()
		cfg.B2AppKey = strings.TrimSpace(c.Scanner.Text())
	}

	if cfg.B2Bucket == "" {
		fmt.Println("Backblaze B2 Bucket not specified in configuration.")
		fmt.Println("Enter Backblaze B2 Bucket name:")
		fmt.Print("> ")
		c.Scanner.Scan()
		cfg.B2Bucket = strings.TrimSpace(c.Scanner.Text())
	}

	// Ask if user wants to save these keys to config
	fmt.Println("Save these keys to your configuration file? (y/n) [y]:")
	fmt.Print("> ")
	c.Scanner.Scan()
	saveChoice := strings.ToLower(strings.TrimSpace(c.Scanner.Text()))

	if saveChoice != "n" && saveChoice != "no" {
		return cfg.SaveToFile("config.json")
	}

	return nil
}

// HandleAPIKeyErrors checks for API key errors and prompts for keys if needed
func (c *CLI) HandleAPIKeyErrors(err error, cfg *config.Config) bool {
	// Check if the error is related to missing API keys
	if strings.Contains(err.Error(), "B2 Key ID is required") ||
		strings.Contains(err.Error(), "B2 Application Key is required") ||
		strings.Contains(err.Error(), "B2 Bucket") ||
		strings.Contains(err.Error(), "authentication failed") ||
		strings.Contains(err.Error(), "authorization") {

		fmt.Printf("API key error: %v\n", err)
		fmt.Println("Would you like to enter API keys now? (y/n) [y]:")
		fmt.Print("> ")
		c.Scanner.Scan()
		choice := strings.ToLower(strings.TrimSpace(c.Scanner.Text()))

		if choice != "n" && choice != "no" {
			err := c.PromptAPIKeys(cfg)
			if err != nil {
				fmt.Printf("Error saving configuration: %v\n", err)
				return false
			}
			return true // Keys were updated
		}
	}

	return false // Not an API key error or user declined to update
}
