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

// Start запускает автоматический планировщик генерации задач
func (s *Scheduler) Start(ctx context.Context, interval time.Duration) {
	s.ticker = time.NewTicker(interval)
	defer s.ticker.Stop()

	log.Printf("Планировщик задач запущен с интервалом: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Планировщик задач остановлен")
			return
		case <-s.stopCh:
			log.Println("Планировщик задач остановлен вручную")
			return
		case <-s.ticker.C:
			if err := s.generateTasks(ctx); err != nil {
				log.Printf("Ошибка генерации повторяющихся задач: %v", err)
			}
		}
	}
}

// Stop останавливает планировщик
func (s *Scheduler) Stop() {
	close(s.stopCh)
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

// generateTasks генерирует повторяющиеся задачи, которые должны быть созданы
func (s *Scheduler) generateTasks(ctx context.Context) error {
	now := s.now()
	
	// Получаем все шаблонные задачи, которым нужно создать новые экземпляры
	templateTasks, err := s.getTasksNeedingGeneration(ctx, now)
	if err != nil {
		return fmt.Errorf("failed to get template tasks: %w", err)
	}

	if len(templateTasks) == 0 {
		return nil // Нет задач для генерации
	}

	generatedCount := 0
	for _, template := range templateTasks {
		if template.Periodicity == nil {
			continue
		}

		// Создаём новый экземпляр задачи из шаблона
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

		// Создаём новую задачу
		created, err := s.repo.Create(ctx, newTask)
		if err != nil {
			log.Printf("Не удалось создать повторяющуюся задачу из шаблона %d: %v", template.ID, err)
			continue
		}

		generatedCount++
		log.Printf("Сгенерирована повторяющаяся задача %d из шаблона %d", created.ID, template.ID)

		// Обновляем время следующего выполнения шаблона
		if err := s.updateNextExecution(ctx, &template, now); err != nil {
			log.Printf("Не удалось обновить следующее выполнение для шаблона %d: %v", template.ID, err)
		}
	}

	if generatedCount > 0 {
		log.Printf("Успешно сгенерировано %d повторяющихся задач", generatedCount)
	}

	return nil
}

// getTasksNeedingGeneration возвращает шаблонные задачи, которые должны создать новые экземпляры
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

// updateNextExecution обновляет время следующего выполнения для шаблонной задачи
func (s *Scheduler) updateNextExecution(ctx context.Context, template *taskdomain.Task, now time.Time) error {
	if template.Periodicity == nil {
		return fmt.Errorf("template has no periodicity")
	}

	// Рассчитываем время следующего выполнения
	nextExec := template.Periodicity.CalculateNextExecution(now)
	
	// Проверяем, не выходит ли следующее выполнение за пределы конечной даты
	if template.Periodicity.EndDate != nil && nextExec.After(*template.Periodicity.EndDate) {
		log.Printf("Шаблон %d достиг конечной даты, больше нет выполнений", template.ID)
		return nil
	}

	// Обновляем шаблон с новым временем следующего выполнения
	template.Periodicity.NextExecution = &nextExec
	template.UpdatedAt = now

	_, err := s.repo.Update(ctx, template)
	return err
}

// GenerateOnce выполняет немедленную генерацию задач (для тестирования или ручного запуска)
func (s *Scheduler) GenerateOnce(ctx context.Context) (int, error) {
	err := s.generateTasks(ctx)
	if err != nil {
		return 0, err
	}
	
	// Считаем сгенерированные задачи
	templateTasks, err := s.getTasksNeedingGeneration(ctx, s.now())
	if err != nil {
		return 0, err
	}
	
	return len(templateTasks), nil
}
