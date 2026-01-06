package model

import (
	"time"

	"github.com/google/uuid"
)

// STTRequest represents a speech-to-text request record
type STTRequest struct {
	ID               uuid.UUID              `json:"id"`
	UserID           uuid.UUID              `json:"user_id"`
	AudioURL         string                 `json:"audio_url"`
	AudioFormat      *string                `json:"audio_format,omitempty"`
	AudioDurationMs  *int                   `json:"audio_duration_ms,omitempty"`
	AudioSizeBytes   *int                   `json:"audio_size_bytes,omitempty"`
	Provider         string                 `json:"stt_provider"`
	Language         *string                `json:"language,omitempty"`
	ModelVersion     *string                `json:"model_version,omitempty"`
	Transcript       *string                `json:"transcript,omitempty"`
	Confidence       *float64               `json:"confidence,omitempty"`
	Status           string                 `json:"status"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
	ProcessingTimeMs *int                   `json:"processing_time_ms,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
}

