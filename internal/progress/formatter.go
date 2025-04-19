package progress

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FormatType represents the format type for output
type FormatType string

const (
	// FormatText is the standard human-readable text format
	FormatText FormatType = "text"
	// FormatJSON is the JSON format for machine-readable output
	FormatJSON FormatType = "json"
	// FormatCSV is the CSV format for spreadsheet-friendly output
	FormatCSV FormatType = "csv"
)

// StageInfo represents information about a stage that can be formatted
type StageInfo struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Current     int64   `json:"current"`
	Total       int64   `json:"total"`
	Percentage  float64 `json:"percentage"`
}

// StatsInfo represents statistics that can be formatted
type StatsInfo struct {
	TotalFiles        int64         `json:"total_files"`
	ProcessedFiles    int64         `json:"processed_files"`
	SkippedFiles      int64         `json:"skipped_files"`
	FailedFiles       int64         `json:"failed_files"`
	BytesProcessed    int64         `json:"bytes_processed"`
	BytesTotal        int64         `json:"bytes_total"`
	BytesUploaded     int64         `json:"bytes_uploaded"`
	StartTime         time.Time     `json:"start_time"`
	ElapsedTime       time.Duration `json:"elapsed_time"`
	CurrentPhase      string        `json:"current_phase"`
	ProcessingRate    float64       `json:"processing_rate"`
	UploadSpeed       float64       `json:"upload_speed"`
	EstimatedTimeLeft time.Duration `json:"estimated_time_left"`
	CompletionPercent float64       `json:"completion_percent"`
}

// Formatter formats progress information in different formats
type Formatter struct {
	formatType FormatType
}

// NewFormatter creates a new formatter of the specified type
func NewFormatter(formatType FormatType) *Formatter {
	return &Formatter{
		formatType: formatType,
	}
}

// FormatStats formats tracker statistics in the configured format
func (f *Formatter) FormatStats(tracker *Tracker) string {
	stats := f.getStatsInfo(tracker)

	switch f.formatType {
	case FormatJSON:
		return f.formatStatsJSON(stats)
	case FormatCSV:
		return f.formatStatsCSV(stats)
	default:
		return f.formatStatsText(stats)
	}
}

// FormatStages formats all stages in the configured format
func (f *Formatter) FormatStages(tracker *Tracker) string {
	stages := f.getStageInfos(tracker)

	switch f.formatType {
	case FormatJSON:
		return f.formatStagesJSON(stages)
	case FormatCSV:
		return f.formatStagesCSV(stages)
	default:
		return f.formatStagesText(stages)
	}
}

// FormatStage formats a single stage in the configured format
func (f *Formatter) FormatStage(tracker *Tracker, stageName string) string {
	stage := tracker.GetStage(stageName)
	if stage == nil {
		return fmt.Sprintf("Stage '%s' not found", stageName)
	}

	stageInfo := f.getStageInfo(stage)

	switch f.formatType {
	case FormatJSON:
		return f.formatStageJSON(stageInfo)
	case FormatCSV:
		return f.formatStageCSV(stageInfo)
	default:
		return f.formatStageText(stageInfo)
	}
}

// getStatsInfo extracts stats info from the tracker
func (f *Formatter) getStatsInfo(tracker *Tracker) StatsInfo {
	tracker.Statistics.mu.Lock()
	defer tracker.Statistics.mu.Unlock()

	stats := tracker.Statistics
	elapsedTime := time.Since(stats.StartTime)

	var completionPercent float64
	if stats.TotalFiles > 0 {
		completionPercent = float64(stats.ProcessedFiles) / float64(stats.TotalFiles) * 100
	}

	return StatsInfo{
		TotalFiles:        stats.TotalFiles,
		ProcessedFiles:    stats.ProcessedFiles,
		SkippedFiles:      stats.SkippedFiles,
		FailedFiles:       stats.FailedFiles,
		BytesProcessed:    stats.BytesProcessed,
		BytesTotal:        stats.BytesTotal,
		BytesUploaded:     stats.BytesUploaded,
		StartTime:         stats.StartTime,
		ElapsedTime:       elapsedTime,
		CurrentPhase:      stats.CurrentPhase,
		ProcessingRate:    stats.ProcessingRate,
		UploadSpeed:       stats.UploadSpeed,
		EstimatedTimeLeft: stats.EstimatedTimeLeft,
		CompletionPercent: completionPercent,
	}
}

// getStageInfos extracts info for all stages
func (f *Formatter) getStageInfos(tracker *Tracker) []StageInfo {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	stageInfos := make([]StageInfo, 0, len(tracker.Stages))

	for _, stage := range tracker.Stages {
		stageInfos = append(stageInfos, f.getStageInfo(stage))
	}

	return stageInfos
}

// getStageInfo extracts info for a single stage
func (f *Formatter) getStageInfo(stage *Stage) StageInfo {
	stage.mu.Lock()
	defer stage.mu.Unlock()

	var percentage float64
	if stage.Total > 0 {
		percentage = float64(stage.Current) / float64(stage.Total) * 100
	}

	return StageInfo{
		Name:        stage.Name,
		Description: stage.Description,
		Current:     stage.Current,
		Total:       stage.Total,
		Percentage:  percentage,
	}
}

// Text formatters
func (f *Formatter) formatStatsText(stats StatsInfo) string {
	var sb strings.Builder

	sb.WriteString("\n📊 Backup Process Statistics 📊\n")
	sb.WriteString("==============================\n")
	sb.WriteString(fmt.Sprintf("Current phase: %s\n", stats.CurrentPhase))
	sb.WriteString(fmt.Sprintf("Total files: %d\n", stats.TotalFiles))
	sb.WriteString(fmt.Sprintf("Files processed: %d\n", stats.ProcessedFiles))
	sb.WriteString(fmt.Sprintf("Files skipped: %d\n", stats.SkippedFiles))
	sb.WriteString(fmt.Sprintf("Files failed: %d\n", stats.FailedFiles))
	sb.WriteString(fmt.Sprintf("Total data: %s\n", formatBytes(stats.BytesTotal)))
	sb.WriteString(fmt.Sprintf("Data processed: %s\n", formatBytes(stats.BytesProcessed)))
	sb.WriteString(fmt.Sprintf("Data uploaded: %s\n", formatBytes(stats.BytesUploaded)))
	sb.WriteString(fmt.Sprintf("Time elapsed: %s\n", formatDuration(stats.ElapsedTime)))

	if stats.UploadSpeed > 0 {
		sb.WriteString(fmt.Sprintf("Upload speed: %s/s\n", formatBytes(int64(stats.UploadSpeed))))
	}

	if stats.EstimatedTimeLeft > 0 {
		sb.WriteString(fmt.Sprintf("Estimated time left: %s\n", formatDuration(stats.EstimatedTimeLeft)))
	}

	sb.WriteString(fmt.Sprintf("Overall completion: %.1f%%\n", stats.CompletionPercent))

	return sb.String()
}

func (f *Formatter) formatStagesText(stages []StageInfo) string {
	var sb strings.Builder

	sb.WriteString("\n🔄 Backup Process Stages 🔄\n")
	sb.WriteString("==========================\n")

	for _, stage := range stages {
		sb.WriteString(f.formatStageText(stage))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (f *Formatter) formatStageText(stage StageInfo) string {
	return fmt.Sprintf("%s: %d/%d (%.1f%%)",
		stage.Description,
		stage.Current,
		stage.Total,
		stage.Percentage)
}

// JSON formatters
func (f *Formatter) formatStatsJSON(stats StatsInfo) string {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting stats as JSON: %v", err)
	}
	return string(data)
}

func (f *Formatter) formatStagesJSON(stages []StageInfo) string {
	data, err := json.MarshalIndent(stages, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting stages as JSON: %v", err)
	}
	return string(data)
}

func (f *Formatter) formatStageJSON(stage StageInfo) string {
	data, err := json.MarshalIndent(stage, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting stage as JSON: %v", err)
	}
	return string(data)
}

// CSV formatters
func (f *Formatter) formatStatsCSV(stats StatsInfo) string {
	var sb strings.Builder

	// Header
	sb.WriteString("total_files,processed_files,skipped_files,failed_files,bytes_processed,bytes_total,bytes_uploaded,elapsed_seconds,processing_rate,upload_speed,completion_percent\n")

	// Data
	sb.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%.1f,%.1f,%.1f,%.1f\n",
		stats.TotalFiles,
		stats.ProcessedFiles,
		stats.SkippedFiles,
		stats.FailedFiles,
		stats.BytesProcessed,
		stats.BytesTotal,
		stats.BytesUploaded,
		stats.ElapsedTime.Seconds(),
		stats.ProcessingRate,
		stats.UploadSpeed,
		stats.CompletionPercent))

	return sb.String()
}

func (f *Formatter) formatStagesCSV(stages []StageInfo) string {
	var sb strings.Builder

	// Header
	sb.WriteString("name,description,current,total,percentage\n")

	// Data
	for _, stage := range stages {
		sb.WriteString(f.formatStageCSV(stage))
	}

	return sb.String()
}

func (f *Formatter) formatStageCSV(stage StageInfo) string {
	return fmt.Sprintf("%s,%s,%d,%d,%.1f\n",
		escapeCSV(stage.Name),
		escapeCSV(stage.Description),
		stage.Current,
		stage.Total,
		stage.Percentage)
}

// escapeCSV escapes a string for CSV output
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(s, "\"", "\"\""))
	}
	return s
}
