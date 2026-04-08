package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		ScheduledAt: normalized.ScheduledAt,
		Periodicity: normalized.Periodicity,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	// For periodic tasks, create the first instance immediately if scheduled
	if normalized.Periodicity != nil && normalized.Periodicity.IsRecurring() {
		if normalized.ScheduledAt != nil && !normalized.ScheduledAt.IsZero() {
			// Create the first scheduled task instance
			firstInstance := &taskdomain.Task{
				Title:        normalized.Title,
				Description:  normalized.Description,
				Status:       taskdomain.StatusNew,
				ScheduledAt:  normalized.ScheduledAt,
				Periodicity:  &taskdomain.Periodicity{
					Type:         taskdomain.PeriodicityTypeNone,
					ParentTaskID: nil, // Will be set after template creation
					IsTemplate:   false,
				},
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			
			// Create template first
			template, err := s.repo.Create(ctx, model)
			if err != nil {
				return nil, err
			}
			
			// Set parent reference for the first instance
			firstInstance.Periodicity.ParentTaskID = &template.ID
			
			// Create the first instance
			_, err = s.repo.Create(ctx, firstInstance)
			if err != nil {
				return nil, err
			}
			
			return template, nil
		}
	}

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		ScheduledAt: normalized.ScheduledAt,
		Periodicity: normalized.Periodicity,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	// Validate periodicity
	if input.Periodicity != nil {
		if !input.Periodicity.IsValid() {
			return CreateInput{}, fmt.Errorf("%w: invalid periodicity configuration", ErrInvalidInput)
		}

		// For periodic tasks, automatically calculate next execution
		if input.Periodicity.IsRecurring() && input.Periodicity.NextExecution == nil {
			if input.Periodicity.StartDate.IsZero() {
				return CreateInput{}, fmt.Errorf("%w: start date is required for periodic tasks", ErrInvalidInput)
			}
			
			// Calculate next execution from start date
			nextExec := input.Periodicity.CalculateNextExecution(input.Periodicity.StartDate.Add(-24 * time.Hour))
			input.Periodicity.NextExecution = &nextExec
		}
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	// Validate periodicity
	if input.Periodicity != nil && !input.Periodicity.IsValid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid periodicity configuration", ErrInvalidInput)
	}

	return input, nil
}
