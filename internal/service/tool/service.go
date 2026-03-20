package tool

import (
	"context"
	"errors"
	"fmt"

	domain "tool_management_backend/internal/domain/tool"
	"tool_management_backend/internal/repository"
)

type Service interface {
	Create(ctx context.Context, input domain.Tool) (domain.Tool, error)
	List(ctx context.Context, filter domain.Filter) ([]domain.Tool, error)
	GetByID(ctx context.Context, id string) (domain.Tool, error)
	Update(ctx context.Context, id string, input domain.Tool) (domain.Tool, error)
	Delete(ctx context.Context, id string) error
}

type ToolService struct {
	repository repository.ToolRepository
}

func New(repository repository.ToolRepository) *ToolService {
	return &ToolService{repository: repository}
}

func (s *ToolService) Create(ctx context.Context, input domain.Tool) (domain.Tool, error) {
	prepared, err := domain.PrepareForCreate(input)
	if err != nil {
		return domain.Tool{}, err
	}

	return s.repository.Create(ctx, prepared)
}

func (s *ToolService) List(ctx context.Context, filter domain.Filter) ([]domain.Tool, error) {
	return s.repository.List(ctx, filter)
}

func (s *ToolService) GetByID(ctx context.Context, id string) (domain.Tool, error) {
	if err := domain.ParseID(id); err != nil {
		return domain.Tool{}, fmt.Errorf("invalid id: %w", err)
	}

	return s.repository.GetByID(ctx, id)
}

func (s *ToolService) Update(ctx context.Context, id string, input domain.Tool) (domain.Tool, error) {
	if err := domain.ParseID(id); err != nil {
		return domain.Tool{}, fmt.Errorf("invalid id: %w", err)
	}

	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return domain.Tool{}, err
	}

	prepared, err := domain.PrepareForUpdate(current, input)
	if err != nil {
		return domain.Tool{}, err
	}

	return s.repository.Update(ctx, prepared)
}

func (s *ToolService) Delete(ctx context.Context, id string) error {
	if err := domain.ParseID(id); err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	return s.repository.Delete(ctx, id)
}

func IsValidationError(err error) (domain.ValidationError, bool) {
	var validationErr domain.ValidationError
	if errors.As(err, &validationErr) {
		return validationErr, true
	}

	return domain.ValidationError{}, false
}
