package api

import (
	"log"
	"net/http"
	"noteme/internal/model"
	"noteme/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// sttRepo is declared in repository.go (shared across package)

// getSTTHistory handles GET /api/stt/history
func getSTTHistory(c *gin.Context) {
	// Get user_id from query parameter (for MVP, we'll use a default or require it)
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		// For MVP, use a default user ID if not provided
		// In production, this should come from authentication
		userIDStr = c.GetHeader("X-User-ID")
		if userIDStr == "" {
			utils.Error(c, http.StatusBadRequest, "user_id is required (query parameter or X-User-ID header)")
			return
		}
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid user_id format")
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get records from repository
	requests, err := sttRepo.ListByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		log.Printf("Error listing STT history: %v", err)
		utils.Error(c, http.StatusInternalServerError, "failed to retrieve history")
		return
	}

	// Format response
	items := make([]gin.H, 0, len(requests))
	for _, req := range requests {
		item := gin.H{
			"id":         req.ID.String(),
			"created_at": req.CreatedAt,
			"status":     req.Status,
		}

		// Add title
		if req.Title != nil && *req.Title != "" {
			item["title"] = *req.Title
		}

		// Add audio info
		if req.AudioURL != "" {
			item["audio_url"] = req.AudioURL
		}
		if req.AudioFormat != nil {
			item["audio_format"] = *req.AudioFormat
		}
		if req.AudioDurationMs != nil {
			item["audio_duration_ms"] = *req.AudioDurationMs
		}

		// Add transcript preview (first 100 chars)
		if req.Transcript != nil && *req.Transcript != "" {
			transcript := *req.Transcript
			if len(transcript) > 100 {
				transcript = transcript[:100] + "..."
			}
			item["transcript_preview"] = transcript
		}

		items = append(items, item)
	}

	utils.Success(c, gin.H{
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"count":  len(items),
	})
}

// getSTTDetail handles GET /api/stt/:id
func getSTTDetail(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		utils.Error(c, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// Get record from repository
	req, err := sttRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error getting STT detail: %v", err)
		utils.Error(c, http.StatusNotFound, "STT request not found")
		return
	}

	// Build response
	response := gin.H{
		"id":         req.ID.String(),
		"user_id":    req.UserID.String(),
		"audio_url":  req.AudioURL,
		"status":     req.Status,
		"created_at": req.CreatedAt,
	}

	// Add title
	if req.Title != nil && *req.Title != "" {
		response["title"] = *req.Title
	}

	// Add optional fields
	if req.AudioFormat != nil {
		response["audio_format"] = *req.AudioFormat
	}
	if req.AudioDurationMs != nil {
		response["audio_duration_ms"] = *req.AudioDurationMs
	}
	if req.AudioSizeBytes != nil {
		response["audio_size_bytes"] = *req.AudioSizeBytes
	}
	if req.Transcript != nil {
		response["transcript"] = *req.Transcript
	}
	if req.Confidence != nil {
		response["confidence"] = *req.Confidence
	}
	if req.ErrorMessage != nil {
		response["error_message"] = *req.ErrorMessage
	}
	if req.ProcessingTimeMs != nil {
		response["processing_time_ms"] = *req.ProcessingTimeMs
	}
	if req.Language != nil {
		response["language"] = *req.Language
	}

	// Add metadata (including ai_analysis)
	if len(req.Metadata) > 0 {
		response["metadata"] = req.Metadata
	}

	utils.Success(c, response)
}

// UpdateTitleRequest represents the request body for updating title
type UpdateTitleRequest struct {
	Title string `json:"title" binding:"required"`
}

// updateSTTTitle handles PATCH /api/stt/:id/title
func updateSTTTitle(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		utils.Error(c, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	var req UpdateTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "title is required")
		return
	}

	if req.Title == "" {
		utils.Error(c, http.StatusBadRequest, "title cannot be empty")
		return
	}

	// Update title in repository
	if err := sttRepo.UpdateTitle(c.Request.Context(), id, req.Title); err != nil {
		log.Printf("Error updating title: %v", err)
		if err.Error() == "STT request not found or already deleted" {
			utils.Error(c, http.StatusNotFound, "STT request not found or already deleted")
		} else {
			utils.Error(c, http.StatusInternalServerError, "failed to update title")
		}
		return
	}

	log.Printf("Title updated for STT request: %s", id.String())

	utils.Success(c, gin.H{
		"id":      id.String(),
		"title":   req.Title,
		"message": "Title updated successfully",
	})
}

// deleteSTT handles DELETE /api/stt/:id
func deleteSTT(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		utils.Error(c, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid id format")
		return
	}

	// Soft delete in repository
	if err := sttRepo.Delete(c.Request.Context(), id); err != nil {
		log.Printf("Error deleting STT request: %v", err)
		if err.Error() == "STT request not found or already deleted" {
			utils.Error(c, http.StatusNotFound, "STT request not found or already deleted")
		} else {
			utils.Error(c, http.StatusInternalServerError, "failed to delete STT request")
		}
		return
	}

	log.Printf("STT request deleted: %s", id.String())

	utils.Success(c, gin.H{
		"id":      id.String(),
		"status":  "deleted",
		"message": "STT request deleted successfully",
	})
}

// searchSTT handles GET /api/stt/search
func searchSTT(c *gin.Context) {
	// Get user_id from query parameter or header
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		userIDStr = c.GetHeader("X-User-ID")
		if userIDStr == "" {
			utils.Error(c, http.StatusBadRequest, "user_id is required (query parameter or X-User-ID header)")
			return
		}
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid user_id format")
		return
	}

	// Get search query
	searchQuery := c.Query("q")
	if searchQuery == "" {
		utils.Error(c, http.StatusBadRequest, "search query (q) is required")
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	log.Printf("Search request: user=%s, query=%s, limit=%d, offset=%d", userIDStr, searchQuery, limit, offset)

	// Search in repository
	requests, err := sttRepo.Search(c.Request.Context(), userID, searchQuery, limit, offset)
	if err != nil {
		log.Printf("Error searching STT requests: %v", err)
		utils.Error(c, http.StatusInternalServerError, "failed to search")
		return
	}

	// Format response
	items := make([]gin.H, 0, len(requests))
	for _, req := range requests {
		item := gin.H{
			"id":         req.ID.String(),
			"created_at": req.CreatedAt,
			"status":     req.Status,
		}

		// Add title (required field for search)
		if req.Title != nil && *req.Title != "" {
			item["title"] = *req.Title
		}

		// Add audio info
		if req.AudioURL != "" {
			item["audio_url"] = req.AudioURL
		}
		if req.AudioFormat != nil {
			item["audio_format"] = *req.AudioFormat
		}
		if req.AudioDurationMs != nil {
			item["audio_duration_ms"] = *req.AudioDurationMs
		}

		// Add transcript preview (first 100 chars)
		if req.Transcript != nil && *req.Transcript != "" {
			transcript := *req.Transcript
			if len(transcript) > 100 {
				transcript = transcript[:100] + "..."
			}
			item["transcript_preview"] = transcript
		}

		// Add AI analysis summary and action_items if available
		if len(req.Metadata) > 0 {
			if aiAnalysis, ok := req.Metadata["ai_analysis"].(map[string]interface{}); ok {
				if summary, ok := aiAnalysis["summary"].([]interface{}); ok && len(summary) > 0 {
					item["summary"] = summary
				}
				if actionItems, ok := aiAnalysis["action_items"].([]interface{}); ok && len(actionItems) > 0 {
					item["action_items"] = actionItems
				}
			}
		}

		items = append(items, item)
	}

	log.Printf("Search returned %d results", len(items))

	utils.Success(c, gin.H{
		"query":  searchQuery,
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"count":  len(items),
	})
}

// Helper function to create STTRequest from storage.Recording
func createSTTRequestFromRecording(recordingID string, userID uuid.UUID, audioURL string, provider string) *model.STTRequest {
	// Parse recording ID (format: rec_<timestamp>)
	// For MVP, we'll generate a new UUID for database
	id := uuid.New()

	audioFormat := getAudioFormatFromURL(audioURL)

	return &model.STTRequest{
		ID:          id,
		UserID:      userID,
		AudioURL:    audioURL,
		AudioFormat: audioFormat,
		Status:      "processing",
		Provider:    provider,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
	}
}

func getAudioFormatFromURL(url string) *string {
	// Extract format from URL or filename
	// This is a simple implementation
	if url == "" {
		return nil
	}
	// Try to extract extension
	ext := ""
	if len(url) > 4 {
		lastDot := -1
		for i := len(url) - 1; i >= 0; i-- {
			if url[i] == '.' {
				lastDot = i
				break
			}
		}
		if lastDot >= 0 && lastDot < len(url)-1 {
			ext = url[lastDot+1:]
		}
	}
	if ext != "" {
		return &ext
	}
	return nil
}
