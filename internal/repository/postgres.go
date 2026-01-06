package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"noteme/internal/db"
	"noteme/internal/model"
	"time"

	"github.com/google/uuid"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository() STTRepository {
	return &postgresRepository{
		db: db.DB,
	}
}

// Create creates a new STT request record
func (r *postgresRepository) Create(ctx context.Context, req *model.STTRequest) error {
	query := `
		INSERT INTO stt_requests (
			id, user_id, audio_url, audio_format, audio_duration_ms, audio_size_bytes,
			stt_provider, language, model_version, transcript, confidence,
			status, error_message, processing_time_ms, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	// Convert metadata to JSONB
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		req.ID,
		req.UserID,
		req.AudioURL,
		req.AudioFormat,
		req.AudioDurationMs,
		req.AudioSizeBytes,
		req.Provider,
		req.Language,
		req.ModelVersion,
		req.Transcript,
		req.Confidence,
		req.Status,
		req.ErrorMessage,
		req.ProcessingTimeMs,
		metadataJSON,
		req.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create STT request: %w", err)
	}

	return nil
}

// UpdateResult updates the STT result
func (r *postgresRepository) UpdateResult(ctx context.Context, req *model.STTRequest) error {
	// Build update query - only update metadata if provided
	var query string
	var args []interface{}

	if req.Metadata != nil && len(req.Metadata) > 0 {
		// Include metadata update
		query = `
			UPDATE stt_requests
			SET 
				transcript = COALESCE($1, transcript),
				confidence = COALESCE($2, confidence),
				status = COALESCE($3, status),
				error_message = COALESCE($4, error_message),
				processing_time_ms = COALESCE($5, processing_time_ms),
				audio_duration_ms = COALESCE($6, audio_duration_ms),
				audio_size_bytes = COALESCE($7, audio_size_bytes),
				metadata = $8::jsonb
			WHERE id = $9
		`

		// Merge metadata if provided
		var existingMetadata map[string]interface{}
		existingReq, getErr := r.GetByID(ctx, req.ID)
		if getErr == nil && existingReq.Metadata != nil {
			existingMetadata = existingReq.Metadata
		} else {
			existingMetadata = make(map[string]interface{})
		}

		// Merge new metadata with existing
		for k, v := range req.Metadata {
			existingMetadata[k] = v
		}

		metadataJSON, marshalErr := json.Marshal(existingMetadata)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal metadata: %w", marshalErr)
		}

		args = []interface{}{
			req.Transcript,
			req.Confidence,
			req.Status,
			req.ErrorMessage,
			req.ProcessingTimeMs,
			req.AudioDurationMs,
			req.AudioSizeBytes,
			metadataJSON,
			req.ID,
		}
	} else {
		// Don't update metadata
		query = `
			UPDATE stt_requests
			SET 
				transcript = COALESCE($1, transcript),
				confidence = COALESCE($2, confidence),
				status = COALESCE($3, status),
				error_message = COALESCE($4, error_message),
				processing_time_ms = COALESCE($5, processing_time_ms),
				audio_duration_ms = COALESCE($6, audio_duration_ms),
				audio_size_bytes = COALESCE($7, audio_size_bytes)
			WHERE id = $8
		`

		args = []interface{}{
			req.Transcript,
			req.Confidence,
			req.Status,
			req.ErrorMessage,
			req.ProcessingTimeMs,
			req.AudioDurationMs,
			req.AudioSizeBytes,
			req.ID,
		}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update STT request: %w", err)
	}

	return nil
}

// GetByID retrieves an STT request by ID
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.STTRequest, error) {
	query := `
		SELECT 
			id, user_id, audio_url, audio_format, audio_duration_ms, audio_size_bytes,
			stt_provider, language, model_version, transcript, confidence,
			status, error_message, processing_time_ms, metadata, created_at
		FROM stt_requests
		WHERE id = $1
	`

	var req model.STTRequest
	var metadataJSON []byte
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID,
		&req.UserID,
		&req.AudioURL,
		&req.AudioFormat,
		&req.AudioDurationMs,
		&req.AudioSizeBytes,
		&req.Provider,
		&req.Language,
		&req.ModelVersion,
		&req.Transcript,
		&req.Confidence,
		&req.Status,
		&req.ErrorMessage,
		&req.ProcessingTimeMs,
		&metadataJSON,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("STT request not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get STT request: %w", err)
	}

	req.CreatedAt = createdAt

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		req.Metadata = make(map[string]interface{})
	}

	return &req, nil
}

// ListByUser retrieves STT requests for a user with pagination
func (r *postgresRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.STTRequest, error) {
	query := `
		SELECT 
			id, user_id, audio_url, audio_format, audio_duration_ms, audio_size_bytes,
			stt_provider, language, model_version, transcript, confidence,
			status, error_message, processing_time_ms, metadata, created_at
		FROM stt_requests
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query STT requests: %w", err)
	}
	defer rows.Close()

	var requests []model.STTRequest
	for rows.Next() {
		var req model.STTRequest
		var metadataJSON []byte
		var createdAt time.Time

		err := rows.Scan(
			&req.ID,
			&req.UserID,
			&req.AudioURL,
			&req.AudioFormat,
			&req.AudioDurationMs,
			&req.AudioSizeBytes,
			&req.Provider,
			&req.Language,
			&req.ModelVersion,
			&req.Transcript,
			&req.Confidence,
			&req.Status,
			&req.ErrorMessage,
			&req.ProcessingTimeMs,
			&metadataJSON,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan STT request: %w", err)
		}

		req.CreatedAt = createdAt

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &req.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		} else {
			req.Metadata = make(map[string]interface{})
		}

		requests = append(requests, req)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return requests, nil
}
