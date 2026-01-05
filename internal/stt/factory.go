package stt

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// CreateProvider creates an STT provider based on environment configuration
func CreateProvider() (Provider, error) {
	providerName := strings.ToLower(os.Getenv("STT_PROVIDER"))

	// Default to FPT if not specified
	if providerName == "" {
		providerName = "fpt"
		log.Printf("[STT Factory] STT_PROVIDER not set, defaulting to 'fpt'")
	}

	switch providerName {
	case "fpt":
		return createFPTProvider()
	case "google":
		return createGoogleProvider()
	default:
		return nil, fmt.Errorf("unsupported STT provider: %s. Supported: fpt, google", providerName)
	}
}

// createFPTProvider creates an FPT STT provider
func createFPTProvider() (Provider, error) {
	apiKey := os.Getenv("FPT_AI_API_KEY")
	url := os.Getenv("FPT_AI_STT_URL")

	if apiKey == "" {
		return nil, fmt.Errorf("FPT_AI_API_KEY environment variable is not set")
	}

	if url == "" {
		url = "https://api.fpt.ai/hmi/asr/v1"
		log.Printf("[STT Factory] FPT_AI_STT_URL not set, using default: %s", url)
	}

	log.Printf("[STT Factory] Creating FPT STT provider")
	return NewFPTProvider(apiKey, url), nil
}

// createGoogleProvider creates a Google STT provider
// GOOGLE_STT_KEY_FILE can be either:
//   - An API key (39 characters, typically starts with "AIzaSy")
//   - A file path to a JSON key file (e.g., "./keys/google-service-account.json")
//   - A JSON string containing the service account credentials
func createGoogleProvider() (Provider, error) {
	projectID := os.Getenv("GOOGLE_STT_PROJECT_ID")
	keyData := os.Getenv("GOOGLE_STT_KEY_FILE")
	
	// Project ID is optional when using API key
	keyDataTrimmed := strings.TrimSpace(keyData)
	isAPIKey := len(keyDataTrimmed) == 39 && strings.HasPrefix(keyDataTrimmed, "AIzaSy")
	
	if !isAPIKey && projectID == "" {
		return nil, fmt.Errorf("GOOGLE_STT_PROJECT_ID environment variable is required when using service account")
	}
	
	if keyData == "" {
		return nil, fmt.Errorf("GOOGLE_STT_KEY_FILE environment variable is not set. It can be:\n  - An API key (39 characters)\n  - A file path to a JSON key file\n  - A JSON string containing service account credentials")
	}

	if isAPIKey {
		log.Printf("[STT Factory] Creating Google STT provider with API key")
	} else {
		log.Printf("[STT Factory] Creating Google STT provider with project: %s", projectID)
	}
	return NewGoogleProvider(projectID, keyData)
}
