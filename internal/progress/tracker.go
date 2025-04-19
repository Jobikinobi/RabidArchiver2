package progress

import (
	"fmt"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Stats represents the statistics of the backup process
type Stats struct {
	TotalFiles        int64
	ProcessedFiles    int64
	SkippedFiles      int64
	FailedFiles       int64
	BytesProcessed    int64
	BytesTotal        int64
	BytesUploaded     int64
	StartTime         time.Time
	LastUpdateTime    time.Time
	CurrentPhase      string
	ProcessingRate    float64 // files per second
	UploadSpeed       float64 // bytes per second
	EstimatedTimeLeft time.Duration
	mu                sync.Mutex
}

// Stage represents a stage in the archiving process
type Stage struct {
	Name        string
	Description string
	Bar         *progressbar.ProgressBar
	Total       int64
	Current     int64
	mu          sync.Mutex
}

// Tracker manages multiple progress bars and statistics
type Tracker struct {
	Stages     map[string]*Stage
	Statistics *Stats
	mu         sync.Mutex
}

// NewTracker creates a new progress tracker
func NewTracker() *Tracker {
	return &Tracker{
		Stages: make(map[string]*Stage),
		Statistics: &Stats{
			StartTime:      time.Now(),
			LastUpdateTime: time.Now(),
		},
	}
}

// AddStage adds a new stage to the tracker
func (t *Tracker) AddStage(name, description string, total int64) *Stage {
	t.mu.Lock()
	defer t.mu.Unlock()

	bar := progressbar.NewOptions64(
		total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(50),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	stage := &Stage{
		Name:        name,
		Description: description,
		Bar:         bar,
		Total:       total,
	}

	t.Stages[name] = stage
	return stage
}

// GetStage retrieves a stage by name
func (t *Tracker) GetStage(name string) *Stage {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Stages[name]
}

// UpdateStage updates a stage's progress
func (t *Tracker) UpdateStage(name string, current int64) {
	stage := t.GetStage(name)
	if stage == nil {
		return
	}

	stage.mu.Lock()
	defer stage.mu.Unlock()

	// Calculate increment needed
	increment := current - stage.Current
	if increment <= 0 {
		return
	}

	stage.Current = current
	stage.Bar.Add64(increment)

	// Update statistics
	t.Statistics.mu.Lock()
	defer t.Statistics.mu.Unlock()

	t.Statistics.CurrentPhase = name
	now := time.Now()

	// Calculate processing rate
	elapsed := now.Sub(t.Statistics.LastUpdateTime).Seconds()
	if elapsed > 0 {
		t.Statistics.ProcessingRate = float64(increment) / elapsed

		// Calculate estimated time left for this stage
		remainingItems := stage.Total - stage.Current
		if t.Statistics.ProcessingRate > 0 {
			remainingSeconds := float64(remainingItems) / t.Statistics.ProcessingRate
			t.Statistics.EstimatedTimeLeft = time.Duration(remainingSeconds) * time.Second
		}
	}

	t.Statistics.LastUpdateTime = now
}

// IncrementStage increments a stage's progress by a given amount
func (t *Tracker) IncrementStage(name string, increment int64) {
	stage := t.GetStage(name)
	if stage == nil {
		return
	}

	stage.mu.Lock()
	stage.Current += increment
	currentValue := stage.Current
	stage.mu.Unlock()

	t.UpdateStage(name, currentValue)
}

// CompleteStage marks a stage as complete
func (t *Tracker) CompleteStage(name string) {
	stage := t.GetStage(name)
	if stage == nil {
		return
	}

	stage.mu.Lock()
	defer stage.mu.Unlock()

	stage.Bar.Finish()
	fmt.Printf("\nCompleted stage: %s\n", stage.Description)
}

// UpdateFileStats updates file processing statistics
func (t *Tracker) UpdateFileStats(processed, skipped, failed int64, bytesProcessed int64) {
	t.Statistics.mu.Lock()
	defer t.Statistics.mu.Unlock()

	t.Statistics.ProcessedFiles += processed
	t.Statistics.SkippedFiles += skipped
	t.Statistics.FailedFiles += failed
	t.Statistics.BytesProcessed += bytesProcessed

	// Update upload speed
	elapsed := time.Since(t.Statistics.StartTime).Seconds()
	if elapsed > 0 {
		t.Statistics.UploadSpeed = float64(t.Statistics.BytesUploaded) / elapsed
	}
}

// UpdateTotals updates the total counts
func (t *Tracker) UpdateTotals(totalFiles, totalBytes int64) {
	t.Statistics.mu.Lock()
	defer t.Statistics.mu.Unlock()

	t.Statistics.TotalFiles = totalFiles
	t.Statistics.BytesTotal = totalBytes
}

// UpdateUploadStats updates upload statistics
func (t *Tracker) UpdateUploadStats(bytesUploaded int64) {
	t.Statistics.mu.Lock()
	defer t.Statistics.mu.Unlock()

	t.Statistics.BytesUploaded += bytesUploaded

	// Update upload speed
	elapsed := time.Since(t.Statistics.StartTime).Seconds()
	if elapsed > 0 {
		t.Statistics.UploadSpeed = float64(t.Statistics.BytesUploaded) / elapsed
	}
}

// PrintSummary prints a summary of the backup process
func (t *Tracker) PrintSummary() {
	t.Statistics.mu.Lock()
	defer t.Statistics.mu.Unlock()

	elapsed := time.Since(t.Statistics.StartTime)

	fmt.Println("\n📊 Backup Process Summary 📊")
	fmt.Println("==============================")
	fmt.Printf("Total files: %d\n", t.Statistics.TotalFiles)
	fmt.Printf("Files processed: %d\n", t.Statistics.ProcessedFiles)
	fmt.Printf("Files skipped: %d\n", t.Statistics.SkippedFiles)
	fmt.Printf("Files failed: %d\n", t.Statistics.FailedFiles)
	fmt.Printf("Total data: %s\n", formatBytes(t.Statistics.BytesTotal))
	fmt.Printf("Data processed: %s\n", formatBytes(t.Statistics.BytesProcessed))
	fmt.Printf("Data uploaded: %s\n", formatBytes(t.Statistics.BytesUploaded))
	fmt.Printf("Total time: %s\n", formatDuration(elapsed))

	if t.Statistics.UploadSpeed > 0 {
		fmt.Printf("Average upload speed: %s/s\n", formatBytes(int64(t.Statistics.UploadSpeed)))
	}

	// Calculate completion percentage
	if t.Statistics.TotalFiles > 0 {
		percentage := float64(t.Statistics.ProcessedFiles) / float64(t.Statistics.TotalFiles) * 100
		fmt.Printf("Completion: %.1f%%\n", percentage)
	}
}

// Helper function to format bytes in a human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Helper function to format duration in a human-readable format
func formatDuration(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
