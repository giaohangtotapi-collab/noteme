package stt

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

// FPTProvider implements STT using FPT.AI Speech-to-Text API
type FPTProvider struct {
	apiKey string
	url    string
}

// NewFPTProvider creates a new FPT STT provider
func NewFPTProvider(apiKey, url string) *FPTProvider {
	return &FPTProvider{
		apiKey: apiKey,
		url:    url,
	}
}

// Name returns the provider name
func (p *FPTProvider) Name() string {
	return "fpt"
}

// FPTSTTResponse represents FPT.AI STT API response
type FPTSTTResponse struct {
	Hypotheses []struct {
		Utterance  string  `json:"utterance"`
		Confidence float64 `json:"confidence"`
	} `json:"hypotheses"`
	ErrorCode int    `json:"errorCode,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Transcribe sends audio file to FPT.AI STT API and returns transcript
func (p *FPTProvider) Transcribe(audioPath string) (*Result, error) {
	startTime := time.Now()

	// Read audio file
	audioBytes, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	// Log audio file info
	fileExt := filepath.Ext(audioPath)
	log.Printf("[FPT STT] Processing audio file: %s, size: %d bytes, extension: %s",
		audioPath, len(audioBytes), fileExt)

	// Check if audio file is too small (likely empty or corrupted)
	if len(audioBytes) < 1000 {
		return nil, fmt.Errorf("audio file too small (%d bytes), may be empty or corrupted", len(audioBytes))
	}

	// Create request
	req, err := http.NewRequest("POST", p.url, bytes.NewReader(audioBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("api-key", p.apiKey)
	req.Header.Set("Content-Type", "text/plain")

	// Send request with timeout
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to FPT.AI: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log raw response for debugging (first 500 chars)
	responsePreview := string(body)
	if len(responsePreview) > 500 {
		responsePreview = responsePreview[:500] + "..."
	}
	log.Printf("[FPT STT] Response preview: %s", responsePreview)

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		log.Printf("[FPT STT] API error: Status %d, Body: %s", resp.StatusCode, string(body))
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("FPT.AI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var sttResp FPTSTTResponse
	if err := json.Unmarshal(body, &sttResp); err != nil {
		log.Printf("[FPT STT] Failed to parse response. Raw body: %s", string(body))
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("failed to parse FPT.AI response: %w", err)
	}

	// Check for API errors
	if sttResp.ErrorCode != 0 {
		log.Printf("[FPT STT] API error code %d: %s", sttResp.ErrorCode, sttResp.Message)
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("FPT.AI API error %d: %s", sttResp.ErrorCode, sttResp.Message)
	}

	// Check if we have hypotheses
	if len(sttResp.Hypotheses) == 0 {
		log.Printf("[FPT STT] No hypotheses returned. Full response: %s", string(body))
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("no speech detected in audio")
	}

	// Get the first (best) hypothesis
	hyp := sttResp.Hypotheses[0]
	transcript := strings.TrimSpace(hyp.Utterance)
	confidence := hyp.Confidence

	// Empty transcript is not valid
	if transcript == "" {
		log.Printf("[FPT STT] Empty transcript returned")
		return &Result{
			Provider:    p.Name(),
			RawResponse: string(body),
		}, fmt.Errorf("empty transcript returned")
	}

	duration := time.Since(startTime)
	log.Printf("[FPT STT] Transcription successful: confidence=%.2f, length=%d, duration=%v",
		confidence, len(transcript), duration)

	return &Result{
		Transcript:  transcript,
		Confidence:  confidence,
		Provider:    p.Name(),
		RawResponse: string(body),
	}, nil
}
