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

	// UpdateTitle updates the title of an STT request
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) error

	// Delete soft deletes an STT request by setting status to "deleted"
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves an STT request by ID (excludes deleted records)
	GetByID(ctx context.Context, id uuid.UUID) (*model.STTRequest, error)

	// ListByUser retrieves STT requests for a user with pagination (excludes deleted records)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.STTRequest, error)

	// Search searches STT requests by meaning in title, summary, and action_items (excludes deleted records)
	Search(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]model.STTRequest, error)
}

