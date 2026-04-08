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

	// Для периодических задач создать первый экземпляр немедленно, если указано расписание
	if normalized.Periodicity != nil && normalized.Periodicity.IsRecurring() {
		if normalized.ScheduledAt != nil && !normalized.ScheduledAt.IsZero() {
			// Создать первый экземпляр задачи по расписанию
			firstInstance := &taskdomain.Task{
				Title:        normalized.Title,
				Description:  normalized.Description,
				Status:       taskdomain.StatusNew,
				ScheduledAt:  normalized.ScheduledAt,
				Periodicity:  &taskdomain.Periodicity{
					Type:         taskdomain.PeriodicityTypeNone,
					ParentTaskID: nil, // Будет установлено после создания шаблона
					IsTemplate:   false,
				},
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			
			// Сначала создать шаблон
			template, err := s.repo.Create(ctx, model)
			if err != nil {
				return nil, err
			}
			
			// Установить ссылку на родителя для первого экземпляра
			firstInstance.Periodicity.ParentTaskID = &template.ID
			
			// Создать первый экземпляр
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
		return nil, fmt.Errorf("%w: id должен быть положительным", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id должен быть положительным", ErrInvalidInput)
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
		return fmt.Errorf("%w: id должен быть положительным", ErrInvalidInput)
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
		return CreateInput{}, fmt.Errorf("%w: заголовок обязателен", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: неверный статус", ErrInvalidInput)
	}

	// Валидация периодичности
	if input.Periodicity != nil {
		if !input.Periodicity.IsValid() {
			return CreateInput{}, fmt.Errorf("%w: неверная конфигурация периодичности", ErrInvalidInput)
		}

		// Для периодических задач автоматически рассчитать следующее выполнение
		if input.Periodicity.IsRecurring() && input.Periodicity.NextExecution == nil {
			if input.Periodicity.StartDate.IsZero() {
				return CreateInput{}, fmt.Errorf("%w: дата начала обязательна для периодических задач", ErrInvalidInput)
			}
			
			// Рассчитать следующее выполнение от даты начала
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
		return UpdateInput{}, fmt.Errorf("%w: заголовок обязателен", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: неверный статус", ErrInvalidInput)
	}

	// Валидация периодичности
	if input.Periodicity != nil && !input.Periodicity.IsValid() {
		return UpdateInput{}, fmt.Errorf("%w: неверная конфигурация периодичности", ErrInvalidInput)
	}

	return input, nil
}
