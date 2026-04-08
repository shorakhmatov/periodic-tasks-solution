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

// GenerateRecurringTasks creates new tasks based on periodic template tasks
func (s *RecurringService) GenerateRecurringTasks(ctx context.Context) ([]taskdomain.Task, error) {
	now := s.now()
	
	// Get all template tasks that need to generate new instances
	templateTasks, err := s.getTasksNeedingGeneration(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get template tasks: %w", err)
	}

	var generatedTasks []taskdomain.Task
	
	for _, template := range templateTasks {
		if template.Periodicity == nil {
			continue
		}

		// Create a new task instance from the template
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

		// Create the new task
		created, err := s.repo.Create(ctx, newTask)
		if err != nil {
			return nil, fmt.Errorf("failed to create recurring task: %w", err)
		}

		generatedTasks = append(generatedTasks, *created)

		// Update the template's next execution time
		if err := s.updateNextExecution(ctx, &template, now); err != nil {
			return nil, fmt.Errorf("failed to update next execution: %w", err)
		}
	}

	return generatedTasks, nil
}

// getTasksNeedingGeneration returns template tasks that should generate new instances
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

// updateNextExecution updates the next execution time for a template task
func (s *RecurringService) updateNextExecution(ctx context.Context, template *taskdomain.Task, now time.Time) error {
	if template.Periodicity == nil {
		return fmt.Errorf("template has no periodicity")
	}

	// Calculate next execution time
	nextExec := template.Periodicity.CalculateNextExecution(now)
	
	// Check if next execution is beyond end date
	if template.Periodicity.EndDate != nil && nextExec.After(*template.Periodicity.EndDate) {
		// No more executions needed, you might want to deactivate the template here
		return nil
	}

	// Update the template with new next execution time
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
