package progress

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// InteractiveModeConfig defines configuration for interactive mode
type InteractiveModeConfig struct {
	RefreshInterval  time.Duration
	ShowDetailedView bool
	TerminalWidth    int
	TerminalHeight   int
}

// DefaultInteractiveModeConfig returns a default configuration for interactive mode
func DefaultInteractiveModeConfig() InteractiveModeConfig {
	return InteractiveModeConfig{
		RefreshInterval:  250 * time.Millisecond,
		ShowDetailedView: true,
		TerminalWidth:    80,
		TerminalHeight:   25,
	}
}

// InteractiveMode handles interactive display of backup progress
type InteractiveMode struct {
	mu          sync.Mutex
	tracker     *Tracker
	config      InteractiveModeConfig
	stopChan    chan struct{}
	stoppedChan chan struct{}
	running     bool

	// UI components
	gauges         map[string]*widgets.Gauge
	statsTable     *widgets.Table
	infoBox        *widgets.Paragraph
	logBox         *widgets.List
	logs           []string
	lastUpdateTime time.Time
}

// NewInteractiveMode creates a new interactive mode display
func NewInteractiveMode(tracker *Tracker, config InteractiveModeConfig) *InteractiveMode {
	return &InteractiveMode{
		tracker:     tracker,
		config:      config,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
		gauges:      make(map[string]*widgets.Gauge),
		logs:        make([]string, 0, 100),
	}
}

// Start starts the interactive display
func (im *InteractiveMode) Start() error {
	im.mu.Lock()
	if im.running {
		im.mu.Unlock()
		return fmt.Errorf("interactive mode is already running")
	}
	im.running = true
	im.mu.Unlock()

	if err := termui.Init(); err != nil {
		return fmt.Errorf("failed to initialize terminal UI: %w", err)
	}

	// Initialize components
	im.initializeComponents()

	// Start UI update goroutine
	go im.updateLoop()

	// Handle Ctrl+C to exit gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		im.Stop()
	}()

	// Handle terminal UI events
	grid := termui.NewGrid()

	// Get terminal dimensions (width and height)
	width, height := termui.TerminalDimensions()

	// Set rectangle with the dimensions
	grid.SetRect(0, 0, width, height)

	termui.Render(grid)
	uiEvents := termui.PollEvents()

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				im.Stop()
				return nil
			case "r":
				// Toggle detailed view
				im.config.ShowDetailedView = !im.config.ShowDetailedView
				im.initializeComponents()
				termui.Render(grid)
			}
		case <-im.stopChan:
			termui.Close()
			close(im.stoppedChan)
			return nil
		}
	}
}

// Stop stops the interactive display
func (im *InteractiveMode) Stop() {
	im.mu.Lock()
	defer im.mu.Unlock()

	if !im.running {
		return
	}

	close(im.stopChan)
	<-im.stoppedChan
	im.running = false
}

// AddLog adds a log message to the interactive display
func (im *InteractiveMode) AddLog(message string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Add timestamp
	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s", timestamp, message)

	// Add to logs with a maximum capacity
	if len(im.logs) >= 100 {
		im.logs = im.logs[1:]
	}
	im.logs = append(im.logs, logLine)

	// Update log box if running
	if im.running && im.logBox != nil {
		im.logBox.Rows = im.logs
	}
}

// initializeComponents initializes the UI components
func (im *InteractiveMode) initializeComponents() {
	// Get terminal size
	termWidth, termHeight := termui.TerminalDimensions()

	// Create a grid
	grid := termui.NewGrid()
	grid.SetRect(0, 0, termWidth, termHeight)

	// Create info box for overall progress
	im.infoBox = widgets.NewParagraph()
	im.infoBox.Title = "Backup Status"
	im.infoBox.BorderStyle.Fg = termui.ColorCyan

	// Create stats table
	im.statsTable = widgets.NewTable()
	im.statsTable.Title = "Statistics"
	im.statsTable.BorderStyle.Fg = termui.ColorGreen
	im.statsTable.RowSeparator = true
	im.statsTable.FillRow = true
	im.statsTable.Rows = [][]string{
		{"Metric", "Value"},
		{"Files Processed", "0"},
		{"Files Skipped", "0"},
		{"Files Failed", "0"},
		{"Data Processed", "0 B"},
		{"Upload Speed", "0 B/s"},
		{"Elapsed Time", "0s"},
		{"Est. Time Left", "N/A"},
	}
	im.statsTable.ColumnWidths = []int{20, 15}

	// Create log box
	im.logBox = widgets.NewList()
	im.logBox.Title = "Activity Log"
	im.logBox.BorderStyle.Fg = termui.ColorYellow
	im.logBox.Rows = im.logs

	// Initialize gauges for each stage
	im.gauges = make(map[string]*widgets.Gauge)
	for name, stage := range im.tracker.Stages {
		gauge := widgets.NewGauge()
		gauge.Title = stage.Description
		gauge.Percent = 0
		gauge.BarColor = termui.ColorBlue
		gauge.BorderStyle.Fg = termui.ColorWhite
		gauge.TitleStyle.Fg = termui.ColorCyan
		im.gauges[name] = gauge
	}

	// Set up the layout based on whether detailed view is enabled
	if im.config.ShowDetailedView {
		// Calculate heights based on number of stages and available space
		stageHeight := 3
		totalStagesHeight := len(im.gauges) * stageHeight
		infoHeight := 5
		statsHeight := 10
		logHeight := termHeight - totalStagesHeight - infoHeight - statsHeight

		// Ensure minimum log height
		if logHeight < 5 {
			logHeight = 5
			// Adjust other components if needed
			if totalStagesHeight > (termHeight - logHeight - infoHeight - 5) {
				statsHeight = 5
			}
		}

		// Set up grid layout
		grid.Set(
			termui.NewRow(float64(infoHeight)/float64(termHeight),
				termui.NewCol(1.0, im.infoBox),
			),
		)

		// Add a row for each stage gauge
		for _, gauge := range im.gauges {
			row := termui.NewRow(float64(stageHeight)/float64(termHeight),
				termui.NewCol(1.0, gauge),
			)
			grid.Items = append(grid.Items, &row)
		}

		// Add stats and log sections
		statsRow := termui.NewRow(float64(statsHeight)/float64(termHeight),
			termui.NewCol(1.0, im.statsTable),
		)
		logRow := termui.NewRow(float64(logHeight)/float64(termHeight),
			termui.NewCol(1.0, im.logBox),
		)
		grid.Items = append(grid.Items, &statsRow, &logRow)
	} else {
		// Simple layout with just overall progress and logs
		grid.Set(
			termui.NewRow(0.2,
				termui.NewCol(1.0, im.infoBox),
			),
			termui.NewRow(0.3,
				termui.NewCol(1.0, im.statsTable),
			),
			termui.NewRow(0.5,
				termui.NewCol(1.0, im.logBox),
			),
		)
	}

	// Set the grid as the UI component to render
	termui.Render(grid)
}

// updateLoop periodically updates the UI components
func (im *InteractiveMode) updateLoop() {
	ticker := time.NewTicker(im.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ticker.C:
			im.updateComponents()
			// Recreate grid with current components
			grid := termui.NewGrid()
			width, height := termui.TerminalDimensions()
			grid.SetRect(0, 0, width, height)
			im.initializeComponents()
			termui.Render(grid)
			return
		}
	}
}

// updateComponents updates all UI components with current data
func (im *InteractiveMode) updateComponents() {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Skip if not running
	if !im.running {
		return
	}

	// Get current statistics
	stats := im.tracker.Statistics
	stats.mu.Lock()
	defer stats.mu.Unlock()

	// Update info box with overall status
	var completionPercent float64
	if stats.TotalFiles > 0 {
		completionPercent = float64(stats.ProcessedFiles) / float64(stats.TotalFiles) * 100
	}

	im.infoBox.Text = fmt.Sprintf(
		"Phase: %s\nFiles: %d/%d\nOverall Progress: %.1f%%\nCurrent Speed: %s/s",
		color.CyanString(stats.CurrentPhase),
		stats.ProcessedFiles,
		stats.TotalFiles,
		completionPercent,
		formatBytes(int64(stats.UploadSpeed)),
	)

	// Update stats table
	elapsedTime := time.Since(stats.StartTime)
	im.statsTable.Rows = [][]string{
		{"Metric", "Value"},
		{"Files Processed", fmt.Sprintf("%d", stats.ProcessedFiles)},
		{"Files Skipped", fmt.Sprintf("%d", stats.SkippedFiles)},
		{"Files Failed", fmt.Sprintf("%d", stats.FailedFiles)},
		{"Data Processed", formatBytes(stats.BytesProcessed)},
		{"Data Uploaded", formatBytes(stats.BytesUploaded)},
		{"Upload Speed", fmt.Sprintf("%s/s", formatBytes(int64(stats.UploadSpeed)))},
		{"Elapsed Time", formatDuration(elapsedTime)},
		{"Est. Time Left", formatDuration(stats.EstimatedTimeLeft)},
	}

	// Update gauges for all stages
	for name, stage := range im.tracker.Stages {
		gauge, ok := im.gauges[name]
		if !ok {
			continue
		}

		stage.mu.Lock()
		var percent int
		if stage.Total > 0 {
			percent = int(float64(stage.Current) / float64(stage.Total) * 100)
		}
		label := fmt.Sprintf("%d/%d", stage.Current, stage.Total)
		stage.mu.Unlock()

		gauge.Percent = percent
		gauge.Label = label
	}

	// Update log list
	im.logBox.Rows = im.logs
	im.logBox.ScrollBottom()
}

// PrintToConsole prints the current progress to the console
func (im *InteractiveMode) PrintToConsole() {
	formatter := NewFormatter(FormatText)
	fmt.Println(formatter.FormatStats(im.tracker))
	fmt.Println(formatter.FormatStages(im.tracker))
}

// RunInteractiveMode starts an interactive display for the given tracker
func RunInteractiveMode(tracker *Tracker) error {
	interactive := NewInteractiveMode(tracker, DefaultInteractiveModeConfig())
	interactive.AddLog("Starting interactive mode")
	return interactive.Start()
}
