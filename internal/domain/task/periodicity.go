package task

import (
	"time"
)

type PeriodicityType string

const (
	PeriodicityTypeNone      PeriodicityType = "none"
	PeriodicityTypeDaily     PeriodicityType = "daily"
	PeriodicityTypeMonthly   PeriodicityType = "monthly"
	PeriodicityTypeSpecific  PeriodicityType = "specific"
	PeriodicityTypeEvenOdd   PeriodicityType = "even_odd"
)

type EvenOddType string

const (
	EvenOddTypeEven  EvenOddType = "even"
	EvenOddTypeOdd   EvenOddType = "odd"
)

type Periodicity struct {
	Type           PeriodicityType `json:"type"`
	DailyInterval  int             `json:"daily_interval,omitempty"`        // для ежедневных: каждые N дней
	MonthlyDay     int             `json:"monthly_day,omitempty"`          // для ежемесячных: число месяца (1-30)
	SpecificDates  []time.Time     `json:"specific_dates,omitempty"`       // для конкретных дат
	EvenOddType    EvenOddType     `json:"even_odd_type,omitempty"`        // для чётных/нечётных дней
	StartDate      time.Time       `json:"start_date"`                     // дата начала периодичности
	EndDate        *time.Time      `json:"end_date,omitempty"`             // опциональная дата окончания
	NextExecution  *time.Time      `json:"next_execution,omitempty"`       // когда создать следующую задачу
	ParentTaskID   *int64          `json:"parent_task_id,omitempty"`       // для сгенерированных задач
	IsTemplate     bool            `json:"is_template"`                    // является ли это шаблонной задачей
}

func (p Periodicity) IsValid() bool {
	switch p.Type {
	case PeriodicityTypeNone:
		return true
	case PeriodicityTypeDaily:
		return p.DailyInterval > 0 && p.DailyInterval <= 365 && !p.StartDate.IsZero()
	case PeriodicityTypeMonthly:
		return p.MonthlyDay >= 1 && p.MonthlyDay <= 30 && !p.StartDate.IsZero()
	case PeriodicityTypeSpecific:
		return len(p.SpecificDates) > 0 && !p.StartDate.IsZero()
	case PeriodicityTypeEvenOdd:
		return (p.EvenOddType == EvenOddTypeEven || p.EvenOddType == EvenOddTypeOdd) && !p.StartDate.IsZero()
	default:
		return false
	}
}

func (p Periodicity) ShouldCreateTask(now time.Time) bool {
	if p.Type == PeriodicityTypeNone || p.NextExecution == nil {
		return false
	}

	// Проверить достижена ли дата окончания
	if p.EndDate != nil && now.After(*p.EndDate) {
		return false
	}

	return !now.Before(*p.NextExecution)
}

func (p Periodicity) CalculateNextExecution(lastExecution time.Time) time.Time {
	switch p.Type {
	case PeriodicityTypeDaily:
		return lastExecution.AddDate(0, 0, p.DailyInterval)
	case PeriodicityTypeMonthly:
		next := lastExecution.AddDate(0, 1, 0)
		// Подстроить под правильное число месяца (1-30)
		for next.Day() != p.MonthlyDay && next.Day() <= 30 {
			next = next.AddDate(0, 0, 1)
		}
		// Если вышли за 30-е число, перейти к следующему месяцу
		if next.Day() != p.MonthlyDay {
			next = time.Date(next.Year(), next.Month()+1, p.MonthlyDay, 0, 0, 0, 0, next.Location())
		}
		return next
	case PeriodicityTypeSpecific:
		// Найти следующую конкретную дату после последнего выполнения
		for _, date := range p.SpecificDates {
			if date.After(lastExecution) {
				return date
			}
		}
		// Больше нет конкретных дат
		return time.Time{}
	case PeriodicityTypeEvenOdd:
		next := lastExecution.AddDate(0, 0, 1)
		for {
			day := next.Day()
			isEven := day%2 == 0
			
			if (p.EvenOddType == EvenOddTypeEven && isEven) || 
			   (p.EvenOddType == EvenOddTypeOdd && !isEven) {
				return next
			}
			next = next.AddDate(0, 0, 1)
		}
	default:
		return time.Time{}
	}
}

func (p Periodicity) IsRecurring() bool {
	return p.Type != PeriodicityTypeNone
}
