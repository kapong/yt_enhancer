package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	GeminiAPIKey      string
	GeminiModel       string
	GeminiTemperature float64
	GeminiMaxTokens   int
	DebugMode         bool   `env:"DEBUG_MODE" envDefault:"false"`
	DebugDir          string `env:"DEBUG_DIR" envDefault:"debug"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY environment variable not set")
	}

	// Default parameters
	cfg := &Config{
		GeminiAPIKey:      apiKey,
		GeminiModel:       "gemini-1.5-flash",
		GeminiTemperature: 0.3,
		GeminiMaxTokens:   8192,
	}

	// Override with environment variables if set
	if envModel := os.Getenv("GEMINI_MODEL"); envModel != "" {
		cfg.GeminiModel = envModel
	}

	if envTemp := os.Getenv("GEMINI_TEMPERATURE"); envTemp != "" {
		if t, err := strconv.ParseFloat(envTemp, 64); err == nil {
			cfg.GeminiTemperature = t
		}
	}

	if envMaxTokens := os.Getenv("GEMINI_MAX_TOKENS"); envMaxTokens != "" {
		if mt, err := strconv.Atoi(envMaxTokens); err == nil {
			cfg.GeminiMaxTokens = mt
		}
	}

	return cfg, nil
}

// LoadEnvFile loads environment variables from a .env file
func LoadEnvFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		// If the file doesn't exist, just return without error
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// Skip comments and empty lines
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Split by the first equals sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) > 1 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
			value = value[1 : len(value)-1]
		}

		// Set environment variable if it's not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return nil
}
