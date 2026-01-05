package stt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider implements STT using Google Cloud Speech-to-Text REST API
type GoogleProvider struct {
	projectID  string
	apiKey     string
	keyFile    string
	httpClient *http.Client
	useAPIKey  bool // true if using API key, false if using service account
}

// NewGoogleProvider creates a new Google STT provider
// keyData can be either:
//   - An API key (39 characters, typically starts with "AIzaSy")
//   - A file path to a JSON key file (e.g., "./keys/google-service-account.json")
//   - A JSON string containing the service account credentials
func NewGoogleProvider(projectID, keyData string) (*GoogleProvider, error) {
	keyDataTrimmed := strings.TrimSpace(keyData)
	
	// Check if it's an API key (typically 39 chars, starts with "AIzaSy")
	if len(keyDataTrimmed) == 39 && strings.HasPrefix(keyDataTrimmed, "AIzaSy") {
		log.Printf("[Google STT] Using API key authentication")
		return &GoogleProvider{
			projectID:  projectID,
			apiKey:     keyDataTrimmed,
			httpClient: &http.Client{Timeout: 90 * time.Second},
			useAPIKey:  true,
		}, nil
	}

	// Otherwise, treat as service account (JSON file or JSON string)
	ctx := context.Background()
	var client *http.Client
	var jsonData []byte
	var err error

	if keyDataTrimmed == "" {
		// Try to use default credentials
		creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w. Please set GOOGLE_STT_KEY_FILE", err)
		}
		client = oauth2.NewClient(ctx, creds.TokenSource)
	} else {
		// Check if keyData is a JSON string (starts with {) or a file path
		if strings.HasPrefix(keyDataTrimmed, "{") {
			// It's a JSON string
			log.Printf("[Google STT] Using JSON string from environment variable")
			jsonData = []byte(keyDataTrimmed)
		} else {
			// It's a file path
			log.Printf("[Google STT] Reading key file: %s", keyDataTrimmed)
			jsonData, err = os.ReadFile(keyDataTrimmed)
			if err != nil {
				return nil, fmt.Errorf("failed to read key file '%s': %w", keyDataTrimmed, err)
			}
		}

		// Parse credentials from JSON
		creds, err := google.CredentialsFromJSON(ctx, jsonData, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to create credentials from JSON: %w", err)
		}
		client = oauth2.NewClient(ctx, creds.TokenSource)
	}

	return &GoogleProvider{
		projectID:  projectID,
		keyFile:    keyDataTrimmed,
		httpClient: client,
		useAPIKey:  false,
	}, nil
}

// Name returns the provider name
func (p *GoogleProvider) Name() string {
	return "google"
}

// GoogleSTTRequest represents Google Speech-to-Text API request
type GoogleSTTRequest struct {
	Config GoogleSTTConfig `json:"config"`
	Audio  GoogleSTTAudio  `json:"audio"`
}

// GoogleSTTConfig represents recognition config
type GoogleSTTConfig struct {
	Encoding                   string `json:"encoding"`
	SampleRateHertz            int    `json:"sampleRateHertz"`
	LanguageCode               string `json:"languageCode"`
	EnableAutomaticPunctuation bool   `json:"enableAutomaticPunctuation"`
	Model                      string `json:"model,omitempty"`
	UseEnhanced                bool   `json:"useEnhanced,omitempty"`
}

// GoogleSTTAudio represents audio data
type GoogleSTTAudio struct {
	Content string `json:"content"` // Base64 encoded
}

// GoogleSTTResponse represents Google Speech-to-Text API response
type GoogleSTTResponse struct {
	Results []GoogleSTTResult `json:"results"`
	Error   *GoogleSTTError   `json:"error,omitempty"`
}

// GoogleSTTResult represents a recognition result
type GoogleSTTResult struct {
	Alternatives []GoogleSTTAlternative `json:"alternatives"`
}

// GoogleSTTAlternative represents a transcript alternative
type GoogleSTTAlternative struct {
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
}

// GoogleSTTError represents an API error
type GoogleSTTError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Transcribe transcribes an audio file using Google Cloud Speech-to-Text REST API
func (p *GoogleProvider) Transcribe(audioPath string) (*Result, error) {
	startTime := time.Now()

	// Read audio file
	audioBytes, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	// Log audio file info
	fileExt := filepath.Ext(audioPath)
	log.Printf("[Google STT] Processing audio file: %s, size: %d bytes, extension: %s",
		audioPath, len(audioBytes), fileExt)

	// Check if audio file is too small
	if len(audioBytes) < 1000 {
		return nil, fmt.Errorf("audio file too small (%d bytes), may be empty or corrupted", len(audioBytes))
	}

	// Determine encoding and sample rate based on file extension
	encoding, sampleRate := getGoogleAudioConfig(fileExt)

	// Base64 encode audio
	audioBase64 := base64.StdEncoding.EncodeToString(audioBytes)

	// Prepare request
	reqBody := GoogleSTTRequest{
		Config: GoogleSTTConfig{
			Encoding:                   encoding,
			SampleRateHertz:            sampleRate,
			LanguageCode:               "vi-VN",
			EnableAutomaticPunctuation: true,
			Model:                      "latest_long",
			UseEnhanced:                true,
		},
		Audio: GoogleSTTAudio{
			Content: audioBase64,
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build API URL
	var apiURL string
	if p.useAPIKey {
		// When using API key, use the standard endpoint with key as query parameter
		// Note: API key must have Speech-to-Text API enabled in Google Cloud Console
		apiURL = fmt.Sprintf("https://speech.googleapis.com/v1/speech:recognize?key=%s", p.apiKey)
		log.Printf("[Google STT] Using API key authentication (endpoint: /v1/speech:recognize)")
	} else {
		// When using service account, use project-based URL
		apiURL = fmt.Sprintf("https://speech.googleapis.com/v1/projects/%s:recognize", p.projectID)
		log.Printf("[Google STT] Using service account authentication (endpoint: /v1/projects/%s:recognize)", p.projectID)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	log.Printf("[Google STT] Calling Google Speech-to-Text API...")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("[Google STT] HTTP error: %v", err)
		return &Result{
			Provider: p.Name(),
		}, fmt.Errorf("failed to send request to Google Speech-to-Text: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log raw response for debugging
	responsePreview := string(body)
	if len(responsePreview) > 500 {
		responsePreview = responsePreview[:500] + "..."
	}
	log.Printf("[Google STT] Response preview: %s", responsePreview)

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		var apiErr GoogleSTTError
		if err := json.Unmarshal(body, &apiErr); err == nil {
			log.Printf("[Google STT] API error: Code %d, Status %s, Message: %s", apiErr.Code, apiErr.Status, apiErr.Message)
			return &Result{
				Provider:    p.Name(),
				RawResponse: string(body),
			}, fmt.Errorf("Google Speech-to-Text API error: %s", apiErr.Message)
		}
		log.Printf("[Google STT] API error: Status %d, Body: %s", resp.StatusCode, string(body))
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("Google Speech-to-Text API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var sttResp GoogleSTTResponse
	if err := json.Unmarshal(body, &sttResp); err != nil {
		log.Printf("[Google STT] Failed to parse response. Raw body: %s", string(body))
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("failed to parse Google Speech-to-Text response: %w", err)
	}

	// Check for API errors
	if sttResp.Error != nil {
		log.Printf("[Google STT] API error: Code %d, Status %s, Message: %s", sttResp.Error.Code, sttResp.Error.Status, sttResp.Error.Message)
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("Google Speech-to-Text API error: %s", sttResp.Error.Message)
	}

	// Check if we have results
	if len(sttResp.Results) == 0 {
		log.Printf("[Google STT] No results returned")
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("no speech detected in audio")
	}

	// Get the first result (best alternative)
	result := sttResp.Results[0]
	if len(result.Alternatives) == 0 {
		log.Printf("[Google STT] No alternatives in result")
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("no transcript alternatives returned")
	}

	// Get the best alternative
	alternative := result.Alternatives[0]
	transcript := strings.TrimSpace(alternative.Transcript)
	confidence := alternative.Confidence

	// Empty transcript is not valid
	if transcript == "" {
		log.Printf("[Google STT] Empty transcript returned")
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("empty transcript returned")
	}

	duration := time.Since(startTime)
	log.Printf("[Google STT] Transcription successful: confidence=%.2f, length=%d, duration=%v",
		confidence, len(transcript), duration)

	return &Result{
		Transcript:  transcript,
		Confidence:  confidence,
		Provider:    p.Name(),
		RawResponse: string(body),
	}, nil
}

// getGoogleAudioConfig determines encoding and sample rate based on file extension
func getGoogleAudioConfig(fileExt string) (string, int) {
	ext := strings.ToLower(fileExt)
	switch ext {
	case ".wav":
		return "LINEAR16", 16000
	case ".mp3":
		return "MP3", 44100
	case ".m4a", ".aac":
		return "AAC", 44100
	case ".ogg":
		return "OGG_OPUS", 48000
	case ".flac":
		return "FLAC", 44100
	default:
		// Default to LINEAR16
		return "LINEAR16", 16000
	}
}
