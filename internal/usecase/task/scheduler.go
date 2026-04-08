package task

import (
	"context"
	"fmt"
	"log"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Scheduler struct {
	repo             Repository
	recurringService *RecurringService
	now              func() time.Time
	ticker           *time.Ticker
	stopCh           chan struct{}
}

func NewScheduler(repo Repository, recurringService *RecurringService) *Scheduler {
	return &Scheduler{
		repo:             repo,
		recurringService: recurringService,
		now:              func() time.Time { return time.Now().UTC() },
		stopCh:           make(chan struct{}),
	}
}

// Start begins the automatic task generation scheduler
func (s *Scheduler) Start(ctx context.Context, interval time.Duration) {
	s.ticker = time.NewTicker(interval)
	defer s.ticker.Stop()

	log.Printf("Task scheduler started with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Task scheduler stopped")
			return
		case <-s.stopCh:
			log.Println("Task scheduler stopped manually")
			return
		case <-s.ticker.C:
			if err := s.generateTasks(ctx); err != nil {
				log.Printf("Error generating recurring tasks: %v", err)
			}
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

// generateTasks generates recurring tasks that are due
func (s *Scheduler) generateTasks(ctx context.Context) error {
	now := s.now()
	
	// Get all template tasks that need to generate new instances
	templateTasks, err := s.getTasksNeedingGeneration(ctx, now)
	if err != nil {
		return fmt.Errorf("failed to get template tasks: %w", err)
	}

	if len(templateTasks) == 0 {
		return nil // No tasks to generate
	}

	generatedCount := 0
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
				Type:         taskdomain.PeriodicityTypeNone,
				ParentTaskID: &template.ID,
				IsTemplate:   false,
			},
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		// Create the new task
		created, err := s.repo.Create(ctx, newTask)
		if err != nil {
			log.Printf("Failed to create recurring task from template %d: %v", template.ID, err)
			continue
		}

		generatedCount++
		log.Printf("Generated recurring task %d from template %d", created.ID, template.ID)

		// Update the template's next execution time
		if err := s.updateNextExecution(ctx, &template, now); err != nil {
			log.Printf("Failed to update next execution for template %d: %v", template.ID, err)
		}
	}

	if generatedCount > 0 {
		log.Printf("Successfully generated %d recurring tasks", generatedCount)
	}

	return nil
}

// getTasksNeedingGeneration returns template tasks that should generate new instances
func (s *Scheduler) getTasksNeedingGeneration(ctx context.Context, now time.Time) ([]taskdomain.Task, error) {
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
func (s *Scheduler) updateNextExecution(ctx context.Context, template *taskdomain.Task, now time.Time) error {
	if template.Periodicity == nil {
		return fmt.Errorf("template has no periodicity")
	}

	// Calculate next execution time
	nextExec := template.Periodicity.CalculateNextExecution(now)
	
	// Check if next execution is beyond end date
	if template.Periodicity.EndDate != nil && nextExec.After(*template.Periodicity.EndDate) {
		log.Printf("Template %d has reached end date, no more executions", template.ID)
		return nil
	}

	// Update the template with new next execution time
	template.Periodicity.NextExecution = &nextExec
	template.UpdatedAt = now

	_, err := s.repo.Update(ctx, template)
	return err
}

// GenerateOnce performs immediate task generation (for testing or manual trigger)
func (s *Scheduler) GenerateOnce(ctx context.Context) (int, error) {
	err := s.generateTasks(ctx)
	if err != nil {
		return 0, err
	}
	
	// Count generated tasks
	templateTasks, err := s.getTasksNeedingGeneration(ctx, s.now())
	if err != nil {
		return 0, err
	}
	
	return len(templateTasks), nil
}
