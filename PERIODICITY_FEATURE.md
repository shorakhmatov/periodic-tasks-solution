# Task Periodicity Feature Implementation

## Overview

This document describes the implementation of the task periodicity feature for the medical information system's task tracker module. The feature allows users to define recurring tasks with different periodicity patterns.

## Design Decisions

### 1. Architecture Approach

**Decision**: Template-based task generation
- **Rationale**: Separates periodic task definitions from actual task instances
- **Benefits**: 
  - Clear separation of concerns
  - Easy to modify periodicity without affecting generated tasks
  - Preserves task history and audit trail
  - Allows for complex periodicity patterns

### 2. Periodicity Types

The implementation supports four types of periodicity as required:

#### Daily (Every N-th Day)
```json
{
  "type": "daily",
  "daily_interval": 3,
  "start_date": "2023-10-25T00:00:00Z",
  "is_template": true
}
```
- Creates a task every N days
- Interval limited to 1-365 days for practical reasons
- Automatically calculates next execution time

#### Monthly (Specific Day of Month)
```json
{
  "type": "monthly",
  "monthly_day": 15,
  "start_date": "2023-10-25T00:00:00Z",
  "is_template": true
}
```
- Creates a task on a specific day each month (1-30)
- Limited to day 30 to handle months with fewer days
- Handles month transitions automatically

#### Specific Dates
```json
{
  "type": "specific",
  "specific_dates": ["2023-12-25T00:00:00Z", "2024-01-01T00:00:00Z"],
  "start_date": "2023-10-25T00:00:00Z",
  "is_template": true
}
```
- Creates tasks only on specified dates
- Useful for holidays, specific events, or irregular schedules
- Dates can be in any order

#### Even/Odd Days
```json
{
  "type": "even_odd",
  "even_odd_type": "even",
  "start_date": "2023-10-25T00:00:00Z",
  "is_template": true
}
```
- Creates tasks on even or odd days of the month
- Useful for alternating day schedules
- Automatically handles month transitions

### 3. Database Schema

**Decision**: Store periodicity fields as separate columns
- **Rationale**: 
  - Allows for efficient querying of periodic tasks
  - Enables database-level filtering and indexing
  - Supports complex periodicity patterns without JSON parsing overhead
- **Trade-off**: More columns but better performance and queryability

**Key Fields**:
- `periodicity_type`: Type of periodicity (daily, monthly, specific, even_odd)
- `periodicity_*`: Type-specific fields (interval, day, dates, etc.)
- `periodicity_start_date`: When periodicity begins
- `periodicity_end_date`: Optional end date for periodicity
- `periodicity_next_execution`: When next task should be generated
- `periodicity_parent_task_id`: Links generated tasks to template
- `periodicity_is_template`: Identifies template vs generated tasks

### 4. Task Generation Strategy

**Decision**: Manual generation via API endpoint
- **Rationale**: 
  - Gives control over when tasks are generated
  - Allows for batch processing and optimization
  - Prevents unexpected task creation
  - Enables integration with external schedulers
- **Alternative Considered**: Automatic generation via database triggers
  - Rejected due to complexity and lack of control

### 5. API Design

**New Endpoints**:
- `POST /api/v1/tasks/generate-recurring` - Generate tasks from templates
- `GET /api/v1/tasks/templates` - List all periodic template tasks
- `GET /api/v1/tasks/{id}/children` - List tasks generated from a template

**Enhanced Existing Endpoints**:
- All task endpoints now support `scheduled_at` and `periodicity` fields
- Backward compatible with existing clients

### 6. Validation and Edge Cases

#### Validation Rules:
1. **Daily**: Interval must be 1-365, start date required
2. **Monthly**: Day must be 1-30, start date required
3. **Specific**: At least one date required, start date required
4. **Even/Odd**: Type must be "even" or "odd", start date required

#### Edge Cases Handled:
1. **Month transitions**: Monthly tasks automatically adjust for shorter months
2. **End dates**: Tasks stop generating after end date
3. **Leap years**: Date calculations handle February 29th correctly
4. **Time zones**: All times stored and processed in UTC
5. **Template modification**: Changing periodicity affects future generations only

### 7. Performance Considerations

#### Database Indexes:
- `periodicity_type` - Quick filtering of periodic tasks
- `periodicity_next_execution` - Efficient task generation queries
- `periodicity_parent_task_id` - Fast child task lookups

#### Query Optimization:
- Template tasks identified by `is_template = true`
- Child tasks linked via `parent_task_id`
- Specific date queries use array operations

### 8. Security Considerations

#### Input Validation:
- All periodicity parameters validated before storage
- Date ranges checked for reasonableness
- SQL injection prevention through parameterized queries

#### Access Control:
- No additional permissions required (inherits existing task permissions)
- Template and child tasks follow same authorization rules

### 9. Error Handling

#### Common Errors:
- **Invalid periodicity configuration**: 400 Bad Request
- **Template not found**: 404 Not Found
- **Date calculation errors**: 500 Internal Server Error
- **Database constraints**: 400 Bad Request

#### Recovery:
- Failed task generation doesn't affect template
- Partial failures logged for investigation
- Next execution calculation retried on next generation

## Usage Examples

### Create a Daily Task Template
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
      "start_date": "2023-10-25T00:00:00Z",
      "is_template": true
    }
  }'
```

### Create a Monthly Task Template
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Monthly Report Generation",
    "description": "Generate monthly patient reports",
    "status": "new",
    "scheduled_at": "2023-11-01T10:00:00Z",
    "periodicity": {
      "type": "monthly",
      "monthly_day": 1,
      "start_date": "2023-10-25T00:00:00Z",
      "is_template": true
    }
  }'
```

### Generate Recurring Tasks
```bash
curl -X POST http://localhost:8080/api/v1/tasks/generate-recurring
```

Response:
```json
{
  "generated_tasks": [...],
  "count": 5
}
```

### List Template Tasks
```bash
curl -X GET http://localhost:8080/api/v1/tasks/templates
```

### List Child Tasks
```bash
curl -X GET http://localhost:8080/api/v1/tasks/1/children
```

## Assumptions and Limitations

### Assumptions:
1. Task generation will be triggered by external scheduler (cron job, etc.)
2. Template tasks are not meant to be completed directly
3. Child tasks inherit template properties but are independent after creation
4. Time zone handling is centralized to UTC

### Limitations:
1. Monthly tasks limited to days 1-30 (day 31+ not supported)
2. No built-in scheduling (requires external trigger)
3. No task generation history/tracking beyond parent-child relationship
4. Bulk operations on periodic tasks not optimized

### Future Enhancements:
1. **Automatic scheduling**: Built-in cron-like scheduler
2. **Complex patterns**: Support for "every Monday and Wednesday"
3. **Task dependencies**: Generate tasks based on other task completion
4. **Generation history**: Track when tasks were generated
5. **Bulk operations**: Mass update/delete of periodic tasks
6. **Time zone support**: Per-user time zone handling

## Testing Strategy

### Unit Tests:
1. Periodicity validation logic
2. Date calculation algorithms
3. Template-child relationship handling

### Integration Tests:
1. End-to-end task creation and generation
2. Database constraint validation
3. API endpoint functionality

### Performance Tests:
1. Large-scale template management
2. Bulk task generation performance
3. Database query optimization

## Conclusion

This implementation provides a robust, flexible, and maintainable solution for task periodicity in the medical information system. The template-based approach ensures clear separation of concerns while supporting all required periodicity patterns. The design prioritizes data integrity, performance, and ease of use for medical staff managing recurring tasks.
