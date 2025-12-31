package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FPTSTTResponse represents FPT.AI STT API response
type FPTSTTResponse struct {
	Hypotheses []struct {
		Utterance  string  `json:"utterance"`
		Confidence float64 `json:"confidence"`
	} `json:"hypotheses"`
	ErrorCode int    `json:"errorCode,omitempty"`
	Message   string `json:"message,omitempty"`
}

// transcribeFPT sends audio file to FPT.AI STT API and returns transcript
func transcribeFPT(audioPath string) (string, float64, error) {
	apiKey := os.Getenv("FPT_AI_API_KEY")
	url := os.Getenv("FPT_AI_STT_URL")

	if apiKey == "" {
		return "", 0, fmt.Errorf("FPT_AI_API_KEY environment variable is not set")
	}
	if url == "" {
		return "", 0, fmt.Errorf("FPT_AI_STT_URL environment variable is not set")
	}

	// Read audio file
	audioBytes, err := os.ReadFile(audioPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read audio file: %w", err)
	}

	// Log audio file info
	log.Printf("Processing audio file: %s, size: %d bytes, extension: %s",
		audioPath, len(audioBytes), filepath.Ext(audioPath))

	// Check if audio file is too small (likely empty or corrupted)
	if len(audioBytes) < 1000 {
		return "", 0, fmt.Errorf("audio file too small (%d bytes), may be empty or corrupted", len(audioBytes))
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(audioBytes))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "text/plain")

	// Send request with timeout
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send request to FPT.AI: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log raw response for debugging (first 500 chars)
	responsePreview := string(body)
	if len(responsePreview) > 500 {
		responsePreview = responsePreview[:500] + "..."
	}
	log.Printf("FPT.AI response preview: %s", responsePreview)

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		log.Printf("FPT.AI API error: Status %d, Body: %s", resp.StatusCode, string(body))
		return "", 0, fmt.Errorf("FPT.AI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var sttResp FPTSTTResponse
	if err := json.Unmarshal(body, &sttResp); err != nil {
		log.Printf("Failed to parse FPT.AI response. Raw body: %s", string(body))
		return "", 0, fmt.Errorf("failed to parse FPT.AI response: %w", err)
	}

	// Check for API errors
	if sttResp.ErrorCode != 0 {
		log.Printf("FPT.AI API error code %d: %s", sttResp.ErrorCode, sttResp.Message)
		return "", 0, fmt.Errorf("FPT.AI API error %d: %s", sttResp.ErrorCode, sttResp.Message)
	}

	// Check if we have hypotheses
	if len(sttResp.Hypotheses) == 0 {
		log.Printf("FPT.AI returned no hypotheses. Full response: %s", string(body))
		// Check if response has alternative structure
		var altResp map[string]interface{}
		if err := json.Unmarshal(body, &altResp); err == nil {
			log.Printf("Alternative response structure: %+v", altResp)
		}
		return "", 0, fmt.Errorf("no speech detected in audio")
	}

	// Get the first (best) hypothesis
	hyp := sttResp.Hypotheses[0]
	transcript := strings.TrimSpace(hyp.Utterance)
	confidence := hyp.Confidence

	// Empty transcript is not valid
	if transcript == "" {
		log.Printf("FPT.AI returned empty transcript string")
		return "", 0, fmt.Errorf("empty transcript returned")
	}

	log.Printf("FPT.AI transcription successful: confidence=%.2f, length=%d", confidence, len(transcript))
	return transcript, confidence, nil
}

// getContentType determines MIME type based on file extension
func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".wav":
		return "audio/wav"
	case ".m4a":
		return "audio/m4a"
	case ".mp3":
		return "audio/mpeg"
	case ".aac":
		return "audio/aac"
	case ".ogg":
		return "audio/ogg"
	default:
		// Default to wav for FPT.AI compatibility
		return "audio/wav"
	}
}
