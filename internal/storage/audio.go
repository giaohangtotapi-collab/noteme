package storage

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Recording struct {
	ID          string
	Path        string
	Status      string // uploaded, processing, processed, failed
	Duration    int    // in seconds
	Size        int64  // file size in bytes
	CreatedAt   string
	Transcript  string
	Confidence  float64
	Error       string
}

var (
	recordings = make(map[string]*Recording)
	mu         sync.Mutex
)

// SaveAudio saves uploaded audio file and returns recording ID
func SaveAudio(file *multipart.FileHeader) (string, error) {
	id := fmt.Sprintf("rec_%d", time.Now().UnixNano())
	dst := filepath.Join("uploads", id+"_"+file.Filename)

	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	if err := saveMultipartFile(file, dst); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Get file size
	fileInfo, err := os.Stat(dst)
	var fileSize int64
	if err == nil {
		fileSize = fileInfo.Size()
	}

	mu.Lock()
	recordings[id] = &Recording{
		ID:        id,
		Path:      dst,
		Status:    "uploaded",
		Size:      fileSize,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	mu.Unlock()

	return id, nil
}

// GetRecording retrieves a recording by ID
func GetRecording(id string) (*Recording, bool) {
	mu.Lock()
	defer mu.Unlock()
	rec, ok := recordings[id]
	if !ok {
		return nil, false
	}
	// Return a copy to avoid race conditions
	recCopy := *rec
	return &recCopy, true
}

// UpdateStatus updates the status of a recording
func UpdateStatus(id, status string) {
	mu.Lock()
	defer mu.Unlock()
	if rec, ok := recordings[id]; ok {
		rec.Status = status
	}
}

// UpdateTranscript updates transcript and confidence
func UpdateTranscript(id string, transcript string, confidence float64) {
	mu.Lock()
	defer mu.Unlock()
	if rec, ok := recordings[id]; ok {
		rec.Transcript = transcript
		rec.Confidence = confidence
	}
}

// UpdateError updates error message
func UpdateError(id string, errorMsg string) {
	mu.Lock()
	defer mu.Unlock()
	if rec, ok := recordings[id]; ok {
		rec.Error = errorMsg
	}
}

// UpdateDuration updates recording duration
func UpdateDuration(id string, duration int) {
	mu.Lock()
	defer mu.Unlock()
	if rec, ok := recordings[id]; ok {
		rec.Duration = duration
	}
}

/* helper */
func saveMultipartFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.ReadFrom(src)
	return err
}
