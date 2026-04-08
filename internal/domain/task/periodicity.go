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
	MonthlyDay     int             `json:"monthly_day,omitempty"`          // для ежемесячных: день месяца (1-30)
	SpecificDates  []time.Time     `json:"specific_dates,omitempty"`       // для конкретных дат
	EvenOddType    EvenOddType     `json:"even_odd_type,omitempty"`        // для чётных/нечётных дней
	StartDate      time.Time       `json:"start_date"`                     // когда начинается периодичность
	EndDate        *time.Time      `json:"end_date,omitempty"`             // опциональная дата окончания
	NextExecution  *time.Time      `json:"next_execution,omitempty"`       // когда должна быть создана следующая задача
	ParentTaskID   *int64          `json:"parent_task_id,omitempty"`       // для сгенерированных задач
	IsTemplate     bool            `json:"is_template"`                    // IsValid проверяет, является ли конфигурация периодичности валидной
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

	// Проверяем, достигнут ли конечный срок
	if p.EndDate != nil && now.After(*p.EndDate) {
		return false
	}

	return !now.Before(*p.NextExecution)
}

func (p Periodicity) CalculateNextExecution(from time.Time) time.Time {
	switch p.Type {
	case PeriodicityTypeDaily:
		return from.AddDate(0, 0, p.DailyInterval)
	case PeriodicityTypeMonthly:
		next := from.AddDate(0, 1, 0)
		// Вычисляем правильный день месяца (1-30)
		for next.Day() != p.MonthlyDay && next.Day() <= 30 {
			next = next.AddDate(0, 0, 1)
		}
		// Обрабатываем переходы между месяцами (например, 31 января -> 28/29 февраля)
		if next.Day() != p.MonthlyDay {
			// Если целевой день не существует в месяце, используем последний день
			for next.Day() > p.MonthlyDay {
				next = next.AddDate(0, 0, -1)
			}
		}
		return next
	case PeriodicityTypeSpecific:
		// Находим следующую конкретную дату после последнего выполнения
		for _, date := range p.SpecificDates {
			if date.After(from) {
				return date
			}
		}
		// Нет больше конкретных дат
		return time.Time{}
	case PeriodicityTypeEvenOdd:
		next := from.AddDate(0, 0, 1)
		for {
			// Ищем следующий день, который соответствует требованию чётности/нечётности
			if (p.EvenOddType == EvenOddTypeEven && next.Day()%2 == 0) ||
				(p.EvenOddType == EvenOddTypeOdd && next.Day()%2 == 1) {
				return next
			}
			next = next.AddDate(0, 0, 1)
		}
	default:
		return time.Time{}
	}
}

func (p Periodicity) IsRecurring() bool {
	return p.Type != PeriodicityTypeNone && p.IsTemplate
}
