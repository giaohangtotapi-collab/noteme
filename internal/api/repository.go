package api

import (
	"log"
	"noteme/internal/repository"
)

// sttRepo is the shared STT repository instance
var sttRepo repository.STTRepository

// InitSTTRepository initializes the STT repository
func InitSTTRepository(repo repository.STTRepository) {
	sttRepo = repo
	if repo != nil {
		log.Printf("STT Repository initialized successfully")
	} else {
		log.Printf("Warning: STT Repository is nil")
	}
}

