package domain

import (
	"context"

	"github.com/google/uuid"
)

type Tag struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	Color       *string   `json:"color,omitempty"`
}

type TagRepo interface {
	Create(ctx context.Context, t *Tag) error
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Tag, error)
	AttachToTask(ctx context.Context, taskID, tagID uuid.UUID) error
	DetachFromTask(ctx context.Context, taskID, tagID uuid.UUID) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]Tag, error)
}
