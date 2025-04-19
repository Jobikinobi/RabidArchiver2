package drives

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Drive represents an external drive
type Drive struct {
	Name        string `json:"name"`
	MountPoint  string `json:"mount_point"`
	Size        string `json:"size"`
	FreeSpace   string `json:"free_space"`
	DriveType   string `json:"drive_type"`
	IsRemovable bool   `json:"is_removable"`
}

// ListDrives returns a list of all connected drives
func ListDrives() ([]Drive, error) {
	// For macOS, we use diskutil to list volumes
	if isOSX() {
		return listDrivesOSX()
	}

	// For Linux, use df (not implemented yet)
	if isLinux() {
		return nil, fmt.Errorf("Linux drive listing not implemented yet")
	}

	// For Windows, use wmic (not implemented yet)
	if isWindows() {
		return nil, fmt.Errorf("Windows drive listing not implemented yet")
	}

	return nil, fmt.Errorf("unsupported operating system")
}

// listDrivesOSX lists drives on macOS systems
func listDrivesOSX() ([]Drive, error) {
	// Run diskutil list to get all disks
	cmd := exec.Command("diskutil", "list", "-plist")
	// We're not using the output directly, but checking if the command succeeds
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute diskutil: %w", err)
	}

	// Parse the output to get drives
	var drives []Drive

	// For now, let's use a simpler approach by just listing mounted volumes
	// Get all volumes in /Volumes
	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return nil, fmt.Errorf("failed to read /Volumes directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip the main disk, which is usually a symlink to /
		if entry.Name() == "Macintosh HD" {
			continue
		}

		mountPoint := filepath.Join("/Volumes", entry.Name())

		// Get drive information
		info, err := getDriveInfoOSX(mountPoint)
		if err != nil {
			// Continue with next drive if we can't get info for this one
			continue
		}

		drives = append(drives, info)
	}

	return drives, nil
}

// getDriveInfoOSX gets information about a drive on macOS
func getDriveInfoOSX(mountPoint string) (Drive, error) {
	var drive Drive
	drive.Name = filepath.Base(mountPoint)
	drive.MountPoint = mountPoint
	drive.IsRemovable = true // Assume external by default

	// Get filesystem info
	cmd := exec.Command("df", "-h", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return drive, fmt.Errorf("failed to get drive info: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return drive, fmt.Errorf("unexpected df output format")
	}

	fields := strings.Fields(lines[1])
	if len(fields) >= 4 {
		drive.Size = fields[1]
		drive.FreeSpace = fields[3]
	}

	// Try to determine if it's an external drive
	// On macOS, we use diskutil info to get more details
	cmd = exec.Command("diskutil", "info", mountPoint)
	output, err = cmd.Output()
	if err == nil {
		info := string(output)

		// Look for Type
		if strings.Contains(info, "Type: ") {
			lines := strings.Split(info, "\n")
			for _, line := range lines {
				if strings.Contains(line, "Type:") {
					drive.DriveType = strings.TrimSpace(strings.Split(line, ":")[1])
					break
				}
			}
		}

		// Determine if removable
		drive.IsRemovable = strings.Contains(info, "Removable Media: Yes") ||
			strings.Contains(info, "External: Yes")
	}

	return drive, nil
}

// Helper functions to determine the operating system
func isOSX() bool {
	return os.Getenv("TERM_PROGRAM") == "Apple_Terminal" ||
		os.Getenv("TERM_PROGRAM") == "iTerm.app" ||
		strings.Contains(os.Getenv("HOME"), "/Users/")
}

func isLinux() bool {
	_, err := os.Stat("/proc")
	return err == nil
}

func isWindows() bool {
	_, err := os.Stat("C:\\Windows")
	return err == nil
}
