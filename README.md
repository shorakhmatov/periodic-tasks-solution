# Feature: Periodicity Tasks - Medical Information System

## Overview

Implemented automatic recurring task generation for the medical task tracker module. Medical staff can now set up periodic tasks that automatically create instances according to defined schedules.

## Requirements Satisfied

1. **Add periodicity settings** - Full API support for task periodicity
2. **Automatic task creation** - Tasks are generated automatically based on settings
3. **Four periodicity types**:
   - Daily (every N-th day)
   - Monthly (specific day 1-30)
   - Specific dates
   - Even/Odd days

## Technical Implementation

### Architecture
- **Template-based system**: Separate periodic templates from task instances
- **Automatic scheduler**: Background process generates tasks every 5 minutes
- **Immediate generation**: First instance created when scheduled date provided

### Key Files Added/Modified

**New Files:**
- `internal/domain/task/periodicity.go` - Periodicity logic and validation
- `migrations/0002_add_periodicity_fields.up.sql` - Database schema
- `internal/usecase/task/recurring_service.go` - Task generation service
- `internal/usecase/task/scheduler.go` - Background scheduler

**Modified Files:**
- `internal/domain/task/task.go` - Added periodicity fields
- `internal/repository/postgres/task_repository.go` - Database operations
- `internal/usecase/task/service.go` - Business logic with auto-generation
- `internal/usecase/task/ports.go` - Input/output structures
- `internal/transport/http/handlers/dto.go` - API DTOs
- `internal/transport/http/handlers/task_handler.go` - HTTP handlers
- `internal/transport/http/router.go` - API routes
- `cmd/api/main.go` - Application entry point with scheduler

### Database Schema
Added 12 periodicity columns to tasks table:
- `periodicity_type`, `periodicity_daily_interval`, `periodicity_monthly_day`
- `periodicity_specific_dates`, `periodicity_even_odd_type`
- `periodicity_start_date`, `periodicity_end_date`, `periodicity_next_execution`
- `periodicity_parent_task_id`, `periodicity_is_template`

## API Usage

### Create Periodic Task
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Daily Patient Check-in",
    "description": "Call patients for daily check-in",
    "status": "new",
    "scheduled_at": "2023-10-25T09:00:00Z",
    "periodicity": {
      "type": "daily",
      "daily_interval": 1,
      "start_date": "2023-10-25T00:00:00Z"
    }
  }'
```

**Result**: Template created + first instance generated automatically

### Periodicity Types Examples

**Daily (every 3 days):**
```json
{
  "type": "daily",
  "daily_interval": 3,
  "start_date": "2023-10-25T00:00:00Z"
}
```

**Monthly (15th day):**
```json
{
  "type": "monthly",
  "monthly_day": 15,
  "start_date": "2023-10-25T00:00:00Z"
}
```

**Specific dates:**
```json
{
  "type": "specific",
  "specific_dates": ["2023-12-25T00:00:00Z", "2024-01-01T00:00:00Z"],
  "start_date": "2023-10-25T00:00:00Z"
}
```

**Even days only:**
```json
{
  "type": "even_odd",
  "even_odd_type": "even",
  "start_date": "2023-10-25T00:00:00Z"
}
```

### Additional Endpoints
- `POST /api/v1/tasks/generate-recurring` - Manual task generation
- `GET /api/v1/tasks/templates` - List periodic templates
- `GET /api/v1/tasks/{id}/children` - List generated tasks

## How It Works

1. **Setup**: Medical staff creates task with periodicity settings
2. **Immediate**: First instance created if `scheduled_at` provided
3. **Automatic**: Background scheduler runs every 5 minutes
4. **Generation**: New instances created when due
5. **Tracking**: Template-child relationships maintained

## Edge Cases Handled

- Month transitions (February 30th handled)
- Leap years
- End dates
- Time zones (UTC)
- Invalid configurations

## Assumptions Made

1. **UTC Time Zone**: All calculations in UTC for consistency
2. **Monthly Day Limit**: Limited to day 30 for shorter months
3. **Daily Interval Limit**: 1-365 days for practicality
4. **Template Pattern**: Separate templates from instances for audit trail
5. **Scheduler Interval**: 5 minutes balances responsiveness and performance

## Running the Project

### With Docker (Recommended)
```bash
docker compose up --build
# Service available at http://localhost:8080
```

### Local Setup
```bash
go mod tidy
go run cmd/api/main.go
```

## Testing the Feature

1. Create a periodic task with any type
2. Verify first instance created immediately
3. Wait 5+ minutes for scheduler to run
4. Check for new instances
5. List templates and children to verify relationships

## Design Decisions

1. **Template-based**: Clear separation of definitions and instances
2. **Automatic generation**: Reduces manual intervention
3. **Background scheduler**: Continuous operation without user action
4. **Comprehensive validation**: Prevents invalid configurations
5. **Performance optimized**: Efficient database queries and indexing

## Future Enhancements

- Built-in scheduler triggers
- Complex patterns (every Monday/Wednesday)
- Task dependencies
- Time zone support per user
- Bulk operations

## Repository Structure

```
cmd/api/main.go                          # Application entry point
internal/
  domain/task/
    task.go                              # Task entity
    periodicity.go                       # Periodicity logic
  usecase/task/
    service.go                           # Business logic
    recurring_service.go                 # Task generation
    scheduler.go                         # Background scheduler
    ports.go                             # Interfaces
  repository/postgres/
    task_repository.go                   # Data access
  transport/http/
    handlers/
      dto.go                            # API DTOs
      task_handler.go                   # HTTP handlers
    router.go                            # Routes
migrations/
  0001_create_tasks.up.sql              # Original schema
  0002_add_periodicity_fields.up.sql    # Periodicity schema
```

This implementation provides a robust, automatic recurring task system that integrates seamlessly with the existing medical information system.
