package task

import (
	"context"
	"fmt"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type RecurringService struct {
	repo Repository
	now  func() time.Time
}

func NewRecurringService(repo Repository) *RecurringService {
	return &RecurringService{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// GenerateRecurringTasks создаёт новые задачи на основе периодических шаблонных задач
func (s *RecurringService) GenerateRecurringTasks(ctx context.Context) ([]taskdomain.Task, error) {
	now := s.now()
	
	// Получить все шаблонные задачи, которые нуждаются в генерации новых экземпляров
	templateTasks, err := s.getTasksNeedingGeneration(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить шаблонные задачи: %w", err)
	}

	var generatedTasks []taskdomain.Task
	
	for _, template := range templateTasks {
		if template.Periodicity == nil {
			continue
		}

		// Создать новый экземпляр задачи из шаблона
		newTask := &taskdomain.Task{
			Title:        template.Title,
			Description:  template.Description,
			Status:       taskdomain.StatusNew,
			ScheduledAt:  template.ScheduledAt,
			Periodicity:  &taskdomain.Periodicity{
				Type:           taskdomain.PeriodicityTypeNone,
				ParentTaskID:   &template.ID,
				IsTemplate:     false,
			},
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		// Создать новую задачу
		created, err := s.repo.Create(ctx, newTask)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать повторяющуюся задачу: %w", err)
		}

		generatedTasks = append(generatedTasks, *created)

		// Обновить время следующего выполнения шаблона
		if err := s.updateNextExecution(ctx, &template, now); err != nil {
			return nil, fmt.Errorf("не удалось обновить следующее выполнение: %w", err)
		}
	}

	return generatedTasks, nil
}

// getTasksNeedingGeneration возвращает шаблонные задачи, которые должны сгенерировать новые экземпляры
func (s *RecurringService) getTasksNeedingGeneration(ctx context.Context, now time.Time) ([]taskdomain.Task, error) {
	// For now, we'll get all template tasks and filter in memory
	// In a production system, you might want to add a database query for efficiency
	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var templateTasks []taskdomain.Task
	for _, task := range allTasks {
		if task.Periodicity != nil && 
		   task.Periodicity.IsTemplate && 
		   task.Periodicity.ShouldCreateTask(now) {
			templateTasks = append(templateTasks, task)
		}
	}

	return templateTasks, nil
}

// updateNextExecution обновляет время следующего выполнения для шаблонной задачи
func (s *RecurringService) updateNextExecution(ctx context.Context, template *taskdomain.Task, now time.Time) error {
	if template.Periodicity == nil {
		return fmt.Errorf("у шаблона нет периодичности")
	}

	// Рассчитать время следующего выполнения
	nextExec := template.Periodicity.CalculateNextExecution(now)
	
	// Проверить, не выходит ли следующее выполнение за дату окончания
	if template.Periodicity.EndDate != nil && nextExec.After(*template.Periodicity.EndDate) {
		// Больше нет выполнений после даты окончания
		return nil
	}

	// Обновить шаблон с новым временем следующего выполнения
	template.Periodicity.NextExecution = &nextExec
	template.UpdatedAt = now

	_, err := s.repo.Update(ctx, template)
	return err
}

// GetTemplateTasks returns all periodic template tasks
func (s *RecurringService) GetTemplateTasks(ctx context.Context) ([]taskdomain.Task, error) {
	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var templateTasks []taskdomain.Task
	for _, task := range allTasks {
		if task.Periodicity != nil && task.Periodicity.IsTemplate {
			templateTasks = append(templateTasks, task)
		}
	}

	return templateTasks, nil
}

// GetChildTasks returns all tasks generated from a specific template
func (s *RecurringService) GetChildTasks(ctx context.Context, parentTaskID int64) ([]taskdomain.Task, error) {
	allTasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var childTasks []taskdomain.Task
	for _, task := range allTasks {
		if task.Periodicity != nil && 
		   task.Periodicity.ParentTaskID != nil && 
		   *task.Periodicity.ParentTaskID == parentTaskID {
			childTasks = append(childTasks, task)
		}
	}

	return childTasks, nil
}
