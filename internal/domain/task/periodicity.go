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
	DailyInterval  int             `json:"daily_interval,omitempty"`        // for daily: every N days
	MonthlyDay     int             `json:"monthly_day,omitempty"`          // for monthly: day of month (1-30)
	SpecificDates  []time.Time     `json:"specific_dates,omitempty"`       // for specific dates
	EvenOddType    EvenOddType     `json:"even_odd_type,omitempty"`        // for even/odd days
	StartDate      time.Time       `json:"start_date"`                     // when periodicity starts
	EndDate        *time.Time      `json:"end_date,omitempty"`             // optional end date
	NextExecution  *time.Time      `json:"next_execution,omitempty"`       // when next task should be created
	ParentTaskID   *int64          `json:"parent_task_id,omitempty"`       // for generated tasks
	IsTemplate     bool            `json:"is_template"`                    // whether this is a template task
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

	// Check if end date is reached
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
		// Adjust to the correct day of month (1-30)
		for next.Day() != p.MonthlyDay && next.Day() <= 30 {
			next = next.AddDate(0, 0, 1)
		}
		// If we went beyond day 30, go to next month
		if next.Day() != p.MonthlyDay {
			next = time.Date(next.Year(), next.Month()+1, p.MonthlyDay, 0, 0, 0, 0, next.Location())
		}
		return next
	case PeriodicityTypeSpecific:
		// Find the next specific date after last execution
		for _, date := range p.SpecificDates {
			if date.After(lastExecution) {
				return date
			}
		}
		// No more specific dates
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
