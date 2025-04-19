package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds application configuration and API keys
type Config struct {
	// Backblaze B2 configuration
	B2KeyID   string `json:"b2_key_id"`
	B2AppKey  string `json:"b2_app_key"`
	B2Bucket  string `json:"b2_bucket"`
	B2KeyName string `json:"b2_key_name"`

	// AI model API keys
	AnthropicAPIKey string `json:"anthropic_api_key"`
	OpenAIAPIKey    string `json:"openai_api_key"`
	MistralAPIKey   string `json:"mistral_api_key"`
	GrokAPIKey      string `json:"grok_api_key"`
	GrpetileAPIKey  string `json:"greptile_api_key"`

	// Other service keys
	GithubToken    string `json:"github_token"`
	NeonAPIKey     string `json:"neon_api_key"`
	BraveSearchKey string `json:"brave_search_key"`

	// App configuration
	CostCapUSD float64 `json:"cost_cap_usd"`
	Summarize  string  `json:"summarize"`
	StubMode   string  `json:"stub_mode"`
}

// Default configuration values
var defaults = Config{
	B2Bucket:   "RabidArchiver",
	B2KeyName:  "rabidarchiver",
	CostCapUSD: 5.0,
	Summarize:  "default",
	StubMode:   "webloc",
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := defaults

	// Load B2 configuration
	if keyID := os.Getenv("B2_KEY_ID"); keyID != "" {
		config.B2KeyID = keyID
	}
	if appKey := os.Getenv("B2_APP_KEY"); appKey != "" {
		config.B2AppKey = appKey
	}
	if bucket := os.Getenv("B2_BUCKET"); bucket != "" {
		config.B2Bucket = bucket
	}

	// Load AI model API keys
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		config.AnthropicAPIKey = key
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		config.OpenAIAPIKey = key
	}
	if key := os.Getenv("MISTRAL_API_KEY"); key != "" {
		config.MistralAPIKey = key
	}
	if key := os.Getenv("GROK_API_KEY"); key != "" {
		config.GrokAPIKey = key
	}
	if key := os.Getenv("GREPTILE_API_KEY"); key != "" {
		config.GrpetileAPIKey = key
	}

	// Load other service keys
	if key := os.Getenv("GITHUB_TOKEN"); key != "" {
		config.GithubToken = key
	}
	if key := os.Getenv("NEON_API_KEY"); key != "" {
		config.NeonAPIKey = key
	}
	if key := os.Getenv("BRAVE_SEARCH_KEY"); key != "" {
		config.BraveSearchKey = key
	}

	return &config
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveToFile saves configuration to a JSON file
func (c *Config) SaveToFile(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	// Write to file with strict permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the required configuration is present
func (c *Config) Validate() error {
	if c.B2KeyID == "" {
		return fmt.Errorf("B2 Key ID is required")
	}
	if c.B2AppKey == "" {
		return fmt.Errorf("B2 Application Key is required")
	}
	if c.B2Bucket == "" {
		return fmt.Errorf("B2 Bucket name is required")
	}

	return nil
}

// CreateDefaultConfig creates a default configuration file if it doesn't exist
func CreateDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		// File already exists
		return nil
	}

	config := defaults
	return config.SaveToFile(path)
}
