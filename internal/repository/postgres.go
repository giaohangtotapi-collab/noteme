package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"noteme/internal/db"
	"noteme/internal/model"
	"strings"
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
			stt_provider, language, model_version, title, transcript, confidence,
			status, error_message, processing_time_ms, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
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
		req.Title,
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

	if len(req.Metadata) > 0 {
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
				title = COALESCE(NULLIF($8, ''), title),
				metadata = $9::jsonb
			WHERE id = $10
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
			req.Title,
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
				audio_size_bytes = COALESCE($7, audio_size_bytes),
				title = COALESCE(NULLIF($8, ''), title)
			WHERE id = $9
		`

		args = []interface{}{
			req.Transcript,
			req.Confidence,
			req.Status,
			req.ErrorMessage,
			req.ProcessingTimeMs,
			req.AudioDurationMs,
			req.AudioSizeBytes,
			req.Title,
			req.ID,
		}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update STT request: %w", err)
	}

	return nil
}

// UpdateTitle updates the title of an STT request
func (r *postgresRepository) UpdateTitle(ctx context.Context, id uuid.UUID, title string) error {
	query := `
		UPDATE stt_requests
		SET title = $1
		WHERE id = $2 AND status != 'deleted'
	`

	result, err := r.db.ExecContext(ctx, query, title, id)
	if err != nil {
		return fmt.Errorf("failed to update title: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("STT request not found or already deleted")
	}

	return nil
}

// Delete soft deletes an STT request by setting status to "deleted"
func (r *postgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE stt_requests
		SET status = 'deleted'
		WHERE id = $1 AND status != 'deleted'
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete STT request: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("STT request not found or already deleted")
	}

	return nil
}

// GetByID retrieves an STT request by ID (excludes deleted records)
func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.STTRequest, error) {
	query := `
		SELECT 
			id, user_id, audio_url, audio_format, audio_duration_ms, audio_size_bytes,
			stt_provider, language, model_version, title, transcript, confidence,
			status, error_message, processing_time_ms, metadata, created_at
		FROM stt_requests
		WHERE id = $1 AND status != 'deleted'
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
		&req.Title,
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

// ListByUser retrieves STT requests for a user with pagination (excludes deleted records)
func (r *postgresRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.STTRequest, error) {
	query := `
		SELECT 
			id, user_id, audio_url, audio_format, audio_duration_ms, audio_size_bytes,
			stt_provider, language, model_version, title, transcript, confidence,
			status, error_message, processing_time_ms, metadata, created_at
		FROM stt_requests
		WHERE user_id = $1 AND status != 'deleted'
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
			&req.Title,
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

// Search searches STT requests by meaning in title, summary, and action_items
// Uses ILIKE pattern matching for case-insensitive search
func (r *postgresRepository) Search(ctx context.Context, userID uuid.UUID, searchQuery string, limit, offset int) ([]model.STTRequest, error) {
	// Escape special characters for ILIKE (escape % and _)
	escapedQuery := strings.ReplaceAll(searchQuery, "%", "\\%")
	escapedQuery = strings.ReplaceAll(escapedQuery, "_", "\\_")
	pattern := "%" + escapedQuery + "%"

	query := `
		SELECT DISTINCT
			id, user_id, audio_url, audio_format, audio_duration_ms, audio_size_bytes,
			stt_provider, language, model_version, title, transcript, confidence,
			status, error_message, processing_time_ms, metadata, created_at
		FROM stt_requests
		WHERE user_id = $1 
			AND status != 'deleted'
			AND (
				-- Search in title (required)
				title ILIKE $2
				OR
				-- Search in summary (from metadata.ai_analysis.summary array)
				EXISTS (
					SELECT 1 
					FROM jsonb_array_elements_text(metadata->'ai_analysis'->'summary') AS summary_item
					WHERE summary_item ILIKE $2
				)
				OR
				-- Search in action_items (from metadata.ai_analysis.action_items array)
				EXISTS (
					SELECT 1 
					FROM jsonb_array_elements_text(metadata->'ai_analysis'->'action_items') AS action_item
					WHERE action_item ILIKE $2
				)
			)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, userID, pattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search STT requests: %w", err)
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
			&req.Title,
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
