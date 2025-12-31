package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	FPTApiKey   string
	FPTSTTURL   string
	OpenAIKey   string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		FPTApiKey:   os.Getenv("FPT_AI_API_KEY"),
		FPTSTTURL:   getEnv("FPT_AI_STT_URL", "https://api.fpt.ai/hmi/asr/v1"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
	}

	// Validate required environment variables
	if cfg.FPTApiKey == "" {
		return nil, fmt.Errorf("FPT_AI_API_KEY is required. Please set it as environment variable:\n  Windows PowerShell: $env:FPT_AI_API_KEY=\"your_key\"\n  Windows CMD: set FPT_AI_API_KEY=your_key\n  Linux/Mac: export FPT_AI_API_KEY=\"your_key\"\n\nOr use the setup script: .\\setup.ps1 (PowerShell) or setup.bat (CMD)")
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
