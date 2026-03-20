package repository

import (
	"context"

	domain "tool_management_backend/internal/domain/tool"
)

type ToolRepository interface {
	Create(ctx context.Context, tool domain.Tool) (domain.Tool, error)
	List(ctx context.Context, filter domain.Filter) ([]domain.Tool, error)
	GetByID(ctx context.Context, id string) (domain.Tool, error)
	Update(ctx context.Context, tool domain.Tool) (domain.Tool, error)
	Delete(ctx context.Context, id string) error
}
