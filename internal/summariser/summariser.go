package summariser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Model represents an LLM model
type Model struct {
	Name         string
	Provider     string
	CostPer1KIn  float64
	CostPer1KOut float64
	MaxTokens    int
	Available    bool
}

// SummaryLevel represents the level of summarization
type SummaryLevel string

const (
	// SummaryNone means no summarization
	SummaryNone SummaryLevel = "none"
	// SummaryBasic means basic summarization
	SummaryBasic SummaryLevel = "basic"
	// SummaryDefault means default summarization
	SummaryDefault SummaryLevel = "default"
	// SummaryFull means full summarization
	SummaryFull SummaryLevel = "full"
)

// CostTracker tracks LLM usage costs
type CostTracker struct {
	mu       sync.Mutex
	total    float64
	costCap  float64
	perModel map[string]float64
}

// Config represents the summariser configuration
type Config struct {
	Level       SummaryLevel
	CostCap     float64
	Concurrency int
	Models      []Model
}

// Summary represents a document summary
type Summary struct {
	Title         string
	SourceText    string
	SourceTokens  int
	Summary       string
	SummaryTokens int
	Cost          float64
	Model         string
	CreatedAt     time.Time
}

// Summariser handles text summarization
type Summariser struct {
	config      Config
	costTracker *CostTracker
}

// NewSummariser creates a new summariser
func NewSummariser(config Config) *Summariser {
	// Set default concurrency if not specified
	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}

	// Initialize cost tracker
	costTracker := &CostTracker{
		costCap:  config.CostCap,
		perModel: make(map[string]float64),
	}

	// Check environment variables for API keys and mark models as available
	for i, model := range config.Models {
		switch model.Provider {
		case "openai":
			config.Models[i].Available = os.Getenv("OPENAI_API_KEY") != ""
		case "anthropic":
			config.Models[i].Available = os.Getenv("ANTHROPIC_KEY") != ""
		case "groq":
			config.Models[i].Available = os.Getenv("GROQ_API_KEY") != ""
		case "ollama":
			// Check if ollama is installed
			_, err := os.Stat("/usr/local/bin/ollama")
			config.Models[i].Available = err == nil
		}
	}

	return &Summariser{
		config:      config,
		costTracker: costTracker,
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		Level:       SummaryDefault,
		CostCap:     5.0,
		Concurrency: 2,
		Models: []Model{
			{
				Name:         "llama3-8b-instruct",
				Provider:     "ollama",
				CostPer1KIn:  0.0,
				CostPer1KOut: 0.0,
				MaxTokens:    4096,
			},
			{
				Name:         "llama3-8b-instant",
				Provider:     "groq",
				CostPer1KIn:  0.0001,
				CostPer1KOut: 0.0002,
				MaxTokens:    4096,
			},
			{
				Name:         "claude-haiku",
				Provider:     "anthropic",
				CostPer1KIn:  0.00025,
				CostPer1KOut: 0.00125,
				MaxTokens:    8192,
			},
			{
				Name:         "gpt-4-turbo",
				Provider:     "openai",
				CostPer1KIn:  0.01,
				CostPer1KOut: 0.03,
				MaxTokens:    16384,
			},
		},
	}
}

// Summarise summarizes text
func (s *Summariser) Summarise(ctx context.Context, title, text string) (*Summary, error) {
	if s.config.Level == SummaryNone {
		return &Summary{
			Title:         title,
			SourceText:    text,
			SourceTokens:  estimateTokenCount(text),
			Summary:       "",
			SummaryTokens: 0,
			Cost:          0,
			Model:         "none",
			CreatedAt:     time.Now(),
		}, nil
	}

	// Check if we have any available models
	var availableModels []Model
	for _, model := range s.config.Models {
		if model.Available {
			availableModels = append(availableModels, model)
		}
	}

	if len(availableModels) == 0 {
		return nil, errors.New("no LLM models available for summarization")
	}

	// Check if we're under the cost cap
	if !s.costTracker.CheckBudget(0.01) { // Check with minimum budget
		return nil, fmt.Errorf("cost cap of $%.2f has been reached", s.config.CostCap)
	}

	// Truncate text if it's too long for any model
	maxTokens := 0
	for _, model := range availableModels {
		if model.MaxTokens > maxTokens {
			maxTokens = model.MaxTokens
		}
	}

	sourceTokens := estimateTokenCount(text)
	if sourceTokens > maxTokens-1000 { // Reserve 1000 tokens for output
		text = truncateText(text, maxTokens-1000)
		sourceTokens = estimateTokenCount(text)
	}

	// Use the waterfall approach to find the right model
	// Start with the cheapest model
	sort.Slice(availableModels, func(i, j int) bool {
		return availableModels[i].CostPer1KOut < availableModels[j].CostPer1KOut
	})

	var summary *Summary
	var err error

	for _, model := range availableModels {
		// Calculate expected cost
		expectedCost := calculateCost(text, "", model)

		// Check if we can afford this model
		if !s.costTracker.CheckBudget(expectedCost) {
			continue
		}

		// Try to summarize with this model
		summary, err = s.summarizeWithModel(ctx, title, text, sourceTokens, model)
		if err == nil {
			return summary, nil
		}
	}

	if summary != nil {
		return summary, nil
	}

	return nil, errors.New("failed to summarize text with any available model")
}

// summarizeWithModel summarizes text using a specific model
func (s *Summariser) summarizeWithModel(ctx context.Context, title, text string, sourceTokens int, model Model) (*Summary, error) {
	prompt := buildPrompt(title, text, s.config.Level)

	var summaryText string
	var err error

	switch model.Provider {
	case "ollama":
		summaryText, err = s.summarizeWithOllama(ctx, model.Name, prompt)
	case "groq":
		summaryText, err = s.summarizeWithGroq(ctx, model.Name, prompt)
	case "anthropic":
		summaryText, err = s.summarizeWithAnthropic(ctx, model.Name, prompt)
	case "openai":
		summaryText, err = s.summarizeWithOpenAI(ctx, model.Name, prompt)
	default:
		return nil, fmt.Errorf("unsupported model provider: %s", model.Provider)
	}

	if err != nil {
		return nil, err
	}

	// Calculate actual cost
	summaryTokens := estimateTokenCount(summaryText)
	cost := calculateCost(prompt, summaryText, model)

	// Track cost
	s.costTracker.AddCost(cost, model.Name)

	return &Summary{
		Title:         title,
		SourceText:    text,
		SourceTokens:  sourceTokens,
		Summary:       summaryText,
		SummaryTokens: summaryTokens,
		Cost:          cost,
		Model:         model.Name,
		CreatedAt:     time.Now(),
	}, nil
}

// GetTotalCost returns the total cost incurred
func (s *Summariser) GetTotalCost() float64 {
	return s.costTracker.GetTotal()
}

// GetRemainingBudget returns the remaining budget
func (s *Summariser) GetRemainingBudget() float64 {
	return s.costTracker.GetRemaining()
}

// AddCost adds cost to the tracker
func (c *CostTracker) AddCost(cost float64, model string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.total += cost
	c.perModel[model] += cost
}

// CheckBudget checks if a cost can be accommodated within the budget
func (c *CostTracker) CheckBudget(cost float64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.total+cost <= c.costCap
}

// GetTotal returns the total cost
func (c *CostTracker) GetTotal() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.total
}

// GetRemaining returns the remaining budget
func (c *CostTracker) GetRemaining() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.costCap - c.total
}

// Helper functions

// calculateCost calculates the cost of a request based on input and output tokens
func calculateCost(input, output string, model Model) float64 {
	inputTokens := estimateTokenCount(input)
	outputTokens := estimateTokenCount(output)

	inputCost := float64(inputTokens) * model.CostPer1KIn / 1000
	outputCost := float64(outputTokens) * model.CostPer1KOut / 1000

	return inputCost + outputCost
}

// estimateTokenCount estimates the number of tokens in a text
// This is a very rough estimate; in production, you'd use a proper tokenizer
func estimateTokenCount(text string) int {
	if text == "" {
		return 0
	}
	words := strings.Fields(text)
	return int(float64(len(words)) * 1.3) // Rough estimate: 1 word ≈ 1.3 tokens
}

// truncateText truncates text to approximately maxTokens
func truncateText(text string, maxTokens int) string {
	estimatedWordCount := int(float64(maxTokens) / 1.3)
	words := strings.Fields(text)

	if len(words) <= estimatedWordCount {
		return text
	}

	return strings.Join(words[:estimatedWordCount], " ") + "..."
}

// buildPrompt builds a prompt for the summarization task
func buildPrompt(title, text string, level SummaryLevel) string {
	var instructions string

	switch level {
	case SummaryBasic:
		instructions = "Provide a very brief summary of the main points only. Keep it under 3 sentences."
	case SummaryDefault:
		instructions = "Provide a concise summary that captures the key points and main ideas. Keep it focused and informative."
	case SummaryFull:
		instructions = "Provide a detailed summary that captures all important information, key points, and supporting details."
	default:
		instructions = "Provide a concise summary that captures the key points and main ideas."
	}

	return fmt.Sprintf(`Document Title: %s

Document Text:
%s

Instructions: %s

Summary:`, title, text, instructions)
}

// Implementation of model-specific summarization functions

// summarizeWithOllama summarizes text using Ollama
func (s *Summariser) summarizeWithOllama(ctx context.Context, model, prompt string) (string, error) {
	// In a real implementation, this would call the Ollama API
	// For now, we'll return a placeholder
	return "This is a placeholder summary generated with Ollama. In production, this would call the actual Ollama API to generate a summary.", nil
}

// summarizeWithGroq summarizes text using Groq
func (s *Summariser) summarizeWithGroq(ctx context.Context, model, prompt string) (string, error) {
	// In a real implementation, this would call the Groq API
	// For now, we'll return a placeholder
	return "This is a placeholder summary generated with Groq. In production, this would call the actual Groq API to generate a summary.", nil
}

// summarizeWithAnthropic summarizes text using Anthropic
func (s *Summariser) summarizeWithAnthropic(ctx context.Context, model, prompt string) (string, error) {
	// In a real implementation, this would call the Anthropic API
	// For now, we'll return a placeholder
	return "This is a placeholder summary generated with Anthropic Claude. In production, this would call the actual Anthropic API to generate a summary.", nil
}

// summarizeWithOpenAI summarizes text using OpenAI
func (s *Summariser) summarizeWithOpenAI(ctx context.Context, model, prompt string) (string, error) {
	// In a real implementation, this would call the OpenAI API
	// For now, we'll return a placeholder
	return "This is a placeholder summary generated with OpenAI. In production, this would call the actual OpenAI API to generate a summary.", nil
}
