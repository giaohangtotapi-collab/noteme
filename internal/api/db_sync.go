package api

import (
	"context"
	"log"
	"noteme/internal/ai"
	"noteme/internal/model"
	"noteme/internal/storage"
	"sync"
	"time"

	"github.com/google/uuid"
)

// recordingIDToDBUUIDMap stores mapping between in-memory recordingID and DB UUID
var recordingIDToDBUUIDMap = make(map[string]uuid.UUID)
var mapMu sync.Mutex

// syncToDatabase syncs in-memory recording to database if repository is available
// Returns the DB UUID if successful, nil UUID if failed or skipped
func syncToDatabase(recordingID string, userID uuid.UUID, providerName string) uuid.UUID {
	if sttRepo == nil {
		log.Printf("Warning: sttRepo is nil, skipping database sync for recording %s", recordingID)
		return uuid.Nil // No database, skip
	}

	log.Printf("Syncing recording %s to database (user: %s, provider: %s)", recordingID, userID, providerName)

	rec, ok := storage.GetRecording(recordingID)
	if !ok {
		log.Printf("Warning: Recording %s not found in storage, skipping database sync", recordingID)
		return uuid.Nil
	}

	ctx := context.Background()

	// Check if we already have a DB UUID for this recording
	mapMu.Lock()
	dbUUID, exists := recordingIDToDBUUIDMap[recordingID]
	mapMu.Unlock()

	if exists {
		// Update existing record
		updateReq := &model.STTRequest{
			ID:     dbUUID,
			Status: rec.Status,
		}

		// Set audio_duration_ms (convert from seconds to milliseconds)
		if rec.Duration > 0 {
			durationMs := rec.Duration * 1000
			updateReq.AudioDurationMs = &durationMs
		}

		// Set audio_size_bytes
		if rec.Size > 0 {
			sizeBytes := int(rec.Size)
			updateReq.AudioSizeBytes = &sizeBytes
		}

		// Set transcript and confidence if available
		if rec.Transcript != "" {
			updateReq.Transcript = &rec.Transcript
			updateReq.Confidence = &rec.Confidence
		}

		// Set error message if failed
		if rec.Error != "" {
			updateReq.ErrorMessage = &rec.Error
		}

		if err := sttRepo.UpdateResult(ctx, updateReq); err != nil {
			log.Printf("Warning: Failed to update recording %s in database: %v", recordingID, err)
			return uuid.Nil
		}

		log.Printf("Successfully updated recording %s in database (UUID: %s)", recordingID, dbUUID.String())
		return dbUUID
	}

	// Create new STT request
	sttReq := &model.STTRequest{
		ID:       uuid.New(),
		UserID:   userID,
		AudioURL: rec.Path, // Use local path for MVP
		Status:   rec.Status,
		Provider: providerName,
		Metadata: map[string]interface{}{
			"recording_id": recordingID, // Store mapping in metadata
		},
	}

	// Set audio format
	if rec.Path != "" {
		format := getAudioFormatFromPath(rec.Path)
		if format != nil {
			sttReq.AudioFormat = format
		}
	}

	// Set audio_duration_ms (convert from seconds to milliseconds)
	if rec.Duration > 0 {
		durationMs := rec.Duration * 1000
		sttReq.AudioDurationMs = &durationMs
	}

	// Set audio_size_bytes
	if rec.Size > 0 {
		sizeBytes := int(rec.Size)
		sttReq.AudioSizeBytes = &sizeBytes
	}

	// Set created_at
	sttReq.CreatedAt = time.Now()
	if rec.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, rec.CreatedAt); err == nil {
			sttReq.CreatedAt = t
		}
	}

	// Set transcript and confidence if available
	if rec.Transcript != "" {
		sttReq.Transcript = &rec.Transcript
		sttReq.Confidence = &rec.Confidence
	}

	// Set error message if failed
	if rec.Error != "" {
		sttReq.ErrorMessage = &rec.Error
	}

	// Create record
	if err := sttRepo.Create(ctx, sttReq); err != nil {
		log.Printf("Error: Failed to create recording %s in database: %v", recordingID, err)
		return uuid.Nil
	}

	// Store mapping
	mapMu.Lock()
	recordingIDToDBUUIDMap[recordingID] = sttReq.ID
	mapMu.Unlock()

	log.Printf("Successfully synced recording %s to database with UUID: %s", recordingID, sttReq.ID.String())
	return sttReq.ID
}

// syncAnalysisToDatabase syncs AI analysis to database metadata
func syncAnalysisToDatabase(recordingID string, analysis *ai.AnalysisResult) {
	if sttRepo == nil {
		return // No database, skip
	}

	ctx := context.Background()

	// Get DB UUID from mapping
	mapMu.Lock()
	dbUUID, exists := recordingIDToDBUUIDMap[recordingID]
	mapMu.Unlock()

	if !exists {
		log.Printf("Warning: No DB UUID found for recording %s, skipping analysis sync", recordingID)
		return
	}

	// Build metadata with ai_analysis
	metadata := map[string]interface{}{
		"recording_id": recordingID,
		"ai_analysis": map[string]interface{}{
			"context":      analysis.Context,
			"summary":      analysis.Summary,
			"key_points":   analysis.KeyPoints,
			"action_items": analysis.ActionItems,
			"zalo_brief":   analysis.ZaloBrief,
		},
	}

	// Update metadata and status in database
	updateReq := &model.STTRequest{
		ID:       dbUUID,
		Status:   "success", // Set status to success when analysis completes
		Metadata: metadata,
	}

	if err := sttRepo.UpdateResult(ctx, updateReq); err != nil {
		log.Printf("Warning: Failed to sync analysis for recording %s to database: %v", recordingID, err)
		return
	}

	log.Printf("Synced analysis for recording %s to database with status=success", recordingID)
}

// getDefaultUserID returns a default user ID for MVP
// In production, this should come from authentication
func getDefaultUserID() uuid.UUID {
	// Use a fixed UUID for MVP
	// In production, get from JWT token or session
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

func getAudioFormatFromPath(path string) *string {
	// Extract format from path
	if len(path) < 4 {
		return nil
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			if i < len(path)-1 {
				ext := path[i+1:]
				return &ext
			}
			break
		}
	}
	return nil
}
