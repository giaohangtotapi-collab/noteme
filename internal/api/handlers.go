package api

import (
	"log"
	"net/http"
	"noteme/internal/ai"
	"noteme/internal/storage"
	"noteme/internal/stt"
	"noteme/internal/utils"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	sttProvider     stt.Provider
	sttProviderOnce sync.Once
)

// getSTTProvider returns the STT provider (singleton)
func getSTTProvider() (stt.Provider, error) {
	var err error
	sttProviderOnce.Do(func() {
		sttProvider, err = stt.CreateProvider()
		if err != nil {
			log.Printf("Failed to create STT provider: %v", err)
		} else {
			log.Printf("STT provider initialized: %s", sttProvider.Name())
		}
	})
	return sttProvider, err
}

func RegisterRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", healthCheck)

	// API v1
	v1 := r.Group("/api/v1")
	{
		v1.POST("/recordings", uploadRecording)
		v1.POST("/process/:recording_id", processRecording)
		v1.GET("/recordings/:recording_id", getRecording)
		v1.GET("/recordings/:recording_id/status", getRecordingStatus)
		v1.POST("/ai/analyze/:recording_id", analyzeRecording)
		v1.GET("/ai/analyze/:recording_id", getAnalysis)
		v1.POST("/ai/ask", askAnything)
	}
}

// healthCheck returns server health status
func healthCheck(c *gin.Context) {
	utils.Success(c, gin.H{
		"status":  "ok",
		"service": "noteme-backend",
	})
}

// uploadRecording handles audio file upload
func uploadRecording(c *gin.Context) {
	// Log request info for debugging
	log.Printf("[Upload] Content-Type: %s", c.GetHeader("Content-Type"))
	log.Printf("[Upload] Request method: %s", c.Request.Method)

	// Try to parse multipart form if not already parsed
	if c.Request.MultipartForm == nil {
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
			log.Printf("[Upload] Failed to parse multipart form: %v", err)
			utils.Error(c, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
			return
		}
	}

	// Log all form fields for debugging
	if c.Request.MultipartForm != nil {
		log.Printf("[Upload] Form fields: %v", c.Request.MultipartForm.Value)
		log.Printf("[Upload] Form files: %v", c.Request.MultipartForm.File)
	}

	file, err := c.FormFile("audio_file")
	if err != nil {
		log.Printf("[Upload] FormFile error: %v", err)
		// Try alternative field names
		if file, err = c.FormFile("audio"); err != nil {
			if file, err = c.FormFile("file"); err != nil {
				utils.Error(c, http.StatusBadRequest, "audio_file is required. Error: "+err.Error())
				return
			}
		}
	}

	// Validate file extension
	// iPhone supports: M4A (default), CAF, WAV, AIFF, MP3 (via third-party apps)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".m4a", ".mp3", ".wav", ".aac", ".ogg", ".caf", ".aiff", ".aif"}
	valid := false
	for _, allowed := range allowedExts {
		if ext == allowed {
			valid = true
			break
		}
	}
	if !valid {
		utils.Error(c, http.StatusBadRequest, "unsupported audio format. Supported: m4a, mp3, wav, aac, ogg, caf, aiff")
		return
	}

	// Validate file size (max 25MB)
	if file.Size > 25*1024*1024 {
		utils.Error(c, http.StatusBadRequest, "file size exceeds 25MB limit")
		return
	}

	recordingID, err := storage.SaveAudio(file)
	if err != nil {
		log.Printf("Error saving audio: %v", err)
		utils.Error(c, http.StatusInternalServerError, "failed to save audio file")
		return
	}

	log.Printf("Audio uploaded successfully: %s", recordingID)
	utils.Success(c, gin.H{
		"recording_id": recordingID,
		"status":       "uploaded",
	})
}

// processRecording processes audio file through STT
func processRecording(c *gin.Context) {
	id := c.Param("recording_id")
	if id == "" {
		utils.Error(c, http.StatusBadRequest, "recording_id is required")
		return
	}

	rec, ok := storage.GetRecording(id)
	if !ok {
		utils.Error(c, http.StatusNotFound, "recording not found")
		return
	}

	// Check if already processing or processed
	if rec.Status == "processing" {
		utils.Error(c, http.StatusConflict, "recording is already being processed")
		return
	}

	if rec.Status == "processed" {
		// Return existing transcript if available
		if rec.Transcript != "" {
			utils.Success(c, gin.H{
				"recording_id": id,
				"status":       "processed",
				"language":     "vi",
				"transcript":   rec.Transcript,
				"confidence":   rec.Confidence,
			})
			return
		}
	}

	storage.UpdateStatus(id, "processing")
	log.Printf("Processing recording: %s", id)

	// Get STT provider
	provider, err := getSTTProvider()
	if err != nil {
		log.Printf("STT provider error for recording %s: %v", id, err)
		storage.UpdateStatus(id, "failed")
		storage.UpdateError(id, "STT provider not available: "+err.Error())
		utils.Error(c, http.StatusInternalServerError, "STT provider not available: "+err.Error())
		return
	}

	// Transcribe audio
	result, err := provider.Transcribe(rec.Path)
	if err != nil {
		log.Printf("STT error for recording %s (provider: %s): %v", id, provider.Name(), err)
		storage.UpdateStatus(id, "failed")
		storage.UpdateError(id, err.Error())
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	text := result.Transcript
	conf := result.Confidence
	log.Printf("STT transcription successful (provider: %s): confidence=%.2f, length=%d",
		provider.Name(), conf, len(text))

	// Validate transcript is not empty
	if text == "" {
		log.Printf("Empty transcript for recording %s", id)
		storage.UpdateStatus(id, "failed")
		storage.UpdateError(id, "empty transcript")
		utils.Error(c, http.StatusBadRequest, "no speech detected in audio")
		return
	}

	// Clean transcript with AI (minimize/optimize)
	log.Printf("Cleaning transcript with AI for recording: %s", id)
	cleanedText, err := ai.CleanTranscriptWithAI(text)
	if err != nil {
		log.Printf("Warning: Failed to clean transcript with AI: %v. Using original transcript.", err)
		// Continue with original transcript if cleaning fails
		cleanedText = text
	} else {
		log.Printf("Transcript cleaned successfully. Original: %d chars, Cleaned: %d chars", len(text), len(cleanedText))
	}

	// Update transcript with cleaned version
	storage.UpdateTranscript(id, cleanedText, conf)
	storage.UpdateStatus(id, "processed")
	log.Printf("Recording processed successfully: %s (confidence: %.2f, original length: %d, cleaned length: %d)",
		id, conf, len(text), len(cleanedText))

	utils.Success(c, gin.H{
		"recording_id": id,
		"status":       "processed",
		"language":     "vi",
		"transcript":   cleanedText,
		"confidence":   conf,
	})
}

// getRecording returns recording information
func getRecording(c *gin.Context) {
	id := c.Param("recording_id")
	if id == "" {
		utils.Error(c, http.StatusBadRequest, "recording_id is required")
		return
	}

	rec, ok := storage.GetRecording(id)
	if !ok {
		utils.Error(c, http.StatusNotFound, "recording not found")
		return
	}

	utils.Success(c, gin.H{
		"recording_id": rec.ID,
		"status":       rec.Status,
		"created_at":   rec.CreatedAt,
		"duration":     rec.Duration,
		"transcript":   rec.Transcript,
		"confidence":   rec.Confidence,
	})
}

// getRecordingStatus returns only the status of a recording
func getRecordingStatus(c *gin.Context) {
	id := c.Param("recording_id")
	if id == "" {
		utils.Error(c, http.StatusBadRequest, "recording_id is required")
		return
	}

	rec, ok := storage.GetRecording(id)
	if !ok {
		utils.Error(c, http.StatusNotFound, "recording not found")
		return
	}

	utils.Success(c, gin.H{
		"recording_id": rec.ID,
		"status":       rec.Status,
	})
}

// analyzeRecording analyzes transcript using AI
func analyzeRecording(c *gin.Context) {
	id := c.Param("recording_id")
	if id == "" {
		utils.Error(c, http.StatusBadRequest, "recording_id is required")
		return
	}

	// Get recording
	rec, ok := storage.GetRecording(id)
	if !ok {
		utils.Error(c, http.StatusNotFound, "recording not found")
		return
	}

	// Check if transcript exists
	if rec.Transcript == "" {
		utils.Error(c, http.StatusBadRequest, "transcript not available. Please process recording first")
		return
	}

	// Check if analysis already exists
	if existing, ok := storage.GetAnalysis(id); ok {
		log.Printf("Returning existing analysis for recording: %s", id)
		utils.Success(c, gin.H{
			"recording_id": id,
			"context":      existing.Context,
			"summary":      existing.Summary,
			"action_items": existing.ActionItems,
			"key_points":   existing.KeyPoints,
			"zalo_brief":   existing.ZaloBrief,
		})
		return
	}

	log.Printf("Analyzing recording: %s", id)

	// Detect context
	detectedContext := ai.DetectContext(rec.Transcript)
	log.Printf("Detected context: %s", detectedContext)

	// Analyze transcript
	result, err := ai.AnalyzeTranscript(rec.Transcript, detectedContext)
	if err != nil {
		log.Printf("AI analysis error for recording %s: %v", id, err)
		utils.Error(c, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	// Save analysis
	storage.SaveAnalysis(id, result)
	log.Printf("Analysis saved for recording: %s", id)

	// Return result
	utils.Success(c, gin.H{
		"recording_id": id,
		"context":      result.Context,
		"summary":      result.Summary,
		"action_items": result.ActionItems,
		"key_points":   result.KeyPoints,
		"zalo_brief":   result.ZaloBrief,
	})
}

// getAnalysis retrieves analysis result for a recording
func getAnalysis(c *gin.Context) {
	id := c.Param("recording_id")
	if id == "" {
		utils.Error(c, http.StatusBadRequest, "recording_id is required")
		return
	}

	result, ok := storage.GetAnalysis(id)
	if !ok {
		utils.Error(c, http.StatusNotFound, "analysis not found. Please analyze recording first")
		return
	}

	utils.Success(c, gin.H{
		"recording_id": id,
		"context":      result.Context,
		"summary":      result.Summary,
		"action_items": result.ActionItems,
		"key_points":   result.KeyPoints,
		"zalo_brief":   result.ZaloBrief,
	})
}

// AskRequest represents the ask anything request
type AskRequest struct {
	Question string `json:"question" binding:"required"`
}

// askAnything answers questions based on all analyzed data
func askAnything(c *gin.Context) {
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "question is required")
		return
	}

	if req.Question == "" {
		utils.Error(c, http.StatusBadRequest, "question cannot be empty")
		return
	}

	log.Printf("Ask Anything request: %s", req.Question)

	// Get all analyses
	allAnalyses := storage.GetAllAnalyses()
	if len(allAnalyses) == 0 {
		utils.Error(c, http.StatusBadRequest, "no analysis data available. Please analyze some recordings first")
		return
	}

	log.Printf("Found %d analyses to use as context", len(allAnalyses))

	// Build analysis contexts with recording info
	analysisContexts := make([]ai.AnalysisContext, 0, len(allAnalyses))
	for recordingID, analysis := range allAnalyses {
		// Get recording info for context
		rec, ok := storage.GetRecording(recordingID)
		if !ok {
			// Skip if recording not found, but still use analysis
			analysisContexts = append(analysisContexts, ai.AnalysisContext{
				RecordingID: recordingID,
				Context:     analysis.Context,
				Summary:     analysis.Summary,
				ActionItems: analysis.ActionItems,
				KeyPoints:   analysis.KeyPoints,
			})
			continue
		}

		analysisContexts = append(analysisContexts, ai.AnalysisContext{
			RecordingID: recordingID,
			CreatedAt:   rec.CreatedAt,
			Context:     analysis.Context,
			Summary:     analysis.Summary,
			ActionItems: analysis.ActionItems,
			KeyPoints:   analysis.KeyPoints,
			Transcript:  rec.Transcript,
		})
	}

	// Call AI to answer
	answer, err := ai.AskAnything(req.Question, analysisContexts)
	if err != nil {
		log.Printf("Ask Anything error: %v", err)
		utils.Error(c, http.StatusInternalServerError, "failed to get answer: "+err.Error())
		return
	}

	log.Printf("Ask Anything answer: %s", answer)

	utils.Success(c, gin.H{
		"question": req.Question,
		"answer":   answer,
	})
}
