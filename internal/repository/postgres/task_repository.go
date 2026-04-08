package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (
			title, description, status, scheduled_at,
			periodicity_type, periodicity_daily_interval, periodicity_monthly_day,
			periodicity_specific_dates, periodicity_even_odd_type, periodicity_start_date,
			periodicity_end_date, periodicity_next_execution, periodicity_parent_task_id,
			periodicity_is_template, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, title, description, status, scheduled_at,
			periodicity_type, periodicity_daily_interval, periodicity_monthly_day,
			periodicity_specific_dates, periodicity_even_odd_type, periodicity_start_date,
			periodicity_end_date, periodicity_next_execution, periodicity_parent_task_id,
			periodicity_is_template, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, 
		task.Title, task.Description, task.Status, task.ScheduledAt,
		extractPeriodicityType(task.Periodicity),
		extractDailyInterval(task.Periodicity),
		extractMonthlyDay(task.Periodicity),
		extractSpecificDates(task.Periodicity),
		extractEvenOddType(task.Periodicity),
		extractStartDate(task.Periodicity),
		extractEndDate(task.Periodicity),
		extractNextExecution(task.Periodicity),
		extractParentTaskID(task.Periodicity),
		extractIsTemplate(task.Periodicity),
		task.CreatedAt, task.UpdatedAt)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, scheduled_at,
			periodicity_type, periodicity_daily_interval, periodicity_monthly_day,
			periodicity_specific_dates, periodicity_even_odd_type, periodicity_start_date,
			periodicity_end_date, periodicity_next_execution, periodicity_parent_task_id,
			periodicity_is_template, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			scheduled_at = $4,
			periodicity_type = $5,
			periodicity_daily_interval = $6,
			periodicity_monthly_day = $7,
			periodicity_specific_dates = $8,
			periodicity_even_odd_type = $9,
			periodicity_start_date = $10,
			periodicity_end_date = $11,
			periodicity_next_execution = $12,
			periodicity_parent_task_id = $13,
			periodicity_is_template = $14,
			updated_at = $15
		WHERE id = $16
		RETURNING id, title, description, status, scheduled_at,
			periodicity_type, periodicity_daily_interval, periodicity_monthly_day,
			periodicity_specific_dates, periodicity_even_odd_type, periodicity_start_date,
			periodicity_end_date, periodicity_next_execution, periodicity_parent_task_id,
			periodicity_is_template, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, 
		task.Title, task.Description, task.Status, task.ScheduledAt,
		extractPeriodicityType(task.Periodicity),
		extractDailyInterval(task.Periodicity),
		extractMonthlyDay(task.Periodicity),
		extractSpecificDates(task.Periodicity),
		extractEvenOddType(task.Periodicity),
		extractStartDate(task.Periodicity),
		extractEndDate(task.Periodicity),
		extractNextExecution(task.Periodicity),
		extractParentTaskID(task.Periodicity),
		extractIsTemplate(task.Periodicity),
		task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, scheduled_at,
			periodicity_type, periodicity_daily_interval, periodicity_monthly_day,
			periodicity_specific_dates, periodicity_even_odd_type, periodicity_start_date,
			periodicity_end_date, periodicity_next_execution, periodicity_parent_task_id,
			periodicity_is_template, created_at, updated_at
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task taskdomain.Task
		status string
		periodicityType *string
		dailyInterval *int
		monthlyDay *int
		specificDates []time.Time
		evenOddType *string
		startDate *time.Time
		endDate *time.Time
		nextExecution *time.Time
		parentTaskID *int64
		isTemplate *bool
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.ScheduledAt,
		&periodicityType,
		&dailyInterval,
		&monthlyDay,
		&specificDates,
		&evenOddType,
		&startDate,
		&endDate,
		&nextExecution,
		&parentTaskID,
		&isTemplate,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	// Build periodicity struct if any periodicity fields are set
	if periodicityType != nil {
		task.Periodicity = &taskdomain.Periodicity{
			Type:           taskdomain.PeriodicityType(*periodicityType),
			DailyInterval:  getDailyInterval(dailyInterval),
			MonthlyDay:     getMonthlyDay(monthlyDay),
			SpecificDates:  specificDates,
			EvenOddType:    getEvenOddType(evenOddType),
			StartDate:      getStartDate(startDate),
			EndDate:        endDate,
			NextExecution:  nextExecution,
			ParentTaskID:   parentTaskID,
			IsTemplate:     getIsTemplate(isTemplate),
		}
	}

	return &task, nil
}

// Helper functions for extracting periodicity fields
func extractPeriodicityType(p *taskdomain.Periodicity) *string {
	if p == nil {
		return nil
	}
	return (*string)(&p.Type)
}

func extractDailyInterval(p *taskdomain.Periodicity) *int {
	if p == nil || p.Type != taskdomain.PeriodicityTypeDaily {
		return nil
	}
	return &p.DailyInterval
}

func extractMonthlyDay(p *taskdomain.Periodicity) *int {
	if p == nil || p.Type != taskdomain.PeriodicityTypeMonthly {
		return nil
	}
	return &p.MonthlyDay
}

func extractSpecificDates(p *taskdomain.Periodicity) []time.Time {
	if p == nil || p.Type != taskdomain.PeriodicityTypeSpecific {
		return nil
	}
	return p.SpecificDates
}

func extractEvenOddType(p *taskdomain.Periodicity) *string {
	if p == nil || p.Type != taskdomain.PeriodicityTypeEvenOdd {
		return nil
	}
	return (*string)(&p.EvenOddType)
}

func extractStartDate(p *taskdomain.Periodicity) *time.Time {
	if p == nil || p.Type == taskdomain.PeriodicityTypeNone {
		return nil
	}
	return &p.StartDate
}

func extractEndDate(p *taskdomain.Periodicity) *time.Time {
	if p == nil || p.EndDate == nil {
		return nil
	}
	return p.EndDate
}

func extractNextExecution(p *taskdomain.Periodicity) *time.Time {
	if p == nil || p.NextExecution == nil {
		return nil
	}
	return p.NextExecution
}

func extractParentTaskID(p *taskdomain.Periodicity) *int64 {
	if p == nil || p.ParentTaskID == nil {
		return nil
	}
	return p.ParentTaskID
}

func extractIsTemplate(p *taskdomain.Periodicity) *bool {
	if p == nil {
		return nil
	}
	return &p.IsTemplate
}

// Helper functions for building periodicity from nullable fields
func getDailyInterval(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func getMonthlyDay(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func getEvenOddType(p *string) taskdomain.EvenOddType {
	if p == nil {
		return ""
	}
	return taskdomain.EvenOddType(*p)
}

func getStartDate(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}

func getIsTemplate(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}
