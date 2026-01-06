package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port               string
	FPTApiKey          string
	FPTSTTURL          string
	OpenAIKey          string
	STTProvider        string
	GoogleSTTProjectID string
	GoogleSTTKeyFile   string
	DatabaseURL         string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		FPTApiKey:          os.Getenv("FPT_AI_API_KEY"),
		FPTSTTURL:          getEnv("FPT_AI_STT_URL", "https://api.fpt.ai/hmi/asr/v1"),
		OpenAIKey:          os.Getenv("OPENAI_API_KEY"),
		STTProvider:        getEnv("STT_PROVIDER", "fpt"),
		GoogleSTTProjectID: os.Getenv("GOOGLE_STT_PROJECT_ID"),
		GoogleSTTKeyFile:   os.Getenv("GOOGLE_STT_KEY_FILE"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
	}

	// Validate STT provider configuration
	sttProvider := getEnv("STT_PROVIDER", "fpt")
	if sttProvider == "fpt" {
		if cfg.FPTApiKey == "" {
			return nil, fmt.Errorf("FPT_AI_API_KEY is required when STT_PROVIDER=fpt. Please set it as environment variable:\n  Windows PowerShell: $env:FPT_AI_API_KEY=\"your_key\"\n  Windows CMD: set FPT_AI_API_KEY=your_key\n  Linux/Mac: export FPT_AI_API_KEY=\"your_key\"")
		}
	} else if sttProvider == "google" {
		if cfg.GoogleSTTProjectID == "" {
			return nil, fmt.Errorf("GOOGLE_STT_PROJECT_ID is required when STT_PROVIDER=google")
		}
		if cfg.GoogleSTTKeyFile == "" {
			return nil, fmt.Errorf("GOOGLE_STT_KEY_FILE is required when STT_PROVIDER=google. It can be either:\n  - A file path (e.g., ./keys/google-service-account.json)\n  - A JSON string containing service account credentials")
		}
	}

	// OpenAI key is optional (only needed for AI analysis)
	// Will be validated when AI analysis is called

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
