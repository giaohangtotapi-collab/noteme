package repository

import (
	"context"
	"noteme/internal/model"

	"github.com/google/uuid"
)

// STTRepository defines the interface for STT request data access
type STTRepository interface {
	// Create creates a new STT request record
	Create(ctx context.Context, req *model.STTRequest) error

	// UpdateResult updates the STT result (transcript, confidence, status, etc.)
	UpdateResult(ctx context.Context, req *model.STTRequest) error

	// GetByID retrieves an STT request by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.STTRequest, error)

	// ListByUser retrieves STT requests for a user with pagination
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.STTRequest, error)
}

