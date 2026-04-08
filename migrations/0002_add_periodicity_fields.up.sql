-- Add scheduled_at field for task scheduling
ALTER TABLE tasks ADD COLUMN scheduled_at TIMESTAMPTZ;

-- Add periodicity fields
ALTER TABLE tasks ADD COLUMN periodicity_type TEXT;
ALTER TABLE tasks ADD COLUMN periodicity_daily_interval INTEGER;
ALTER TABLE tasks ADD COLUMN periodicity_monthly_day INTEGER;
ALTER TABLE tasks ADD COLUMN periodicity_specific_dates TIMESTAMPTZ[];
ALTER TABLE tasks ADD COLUMN periodicity_even_odd_type TEXT;
ALTER TABLE tasks ADD COLUMN periodicity_start_date TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN periodicity_end_date TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN periodicity_next_execution TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN periodicity_parent_task_id BIGINT;
ALTER TABLE tasks ADD COLUMN periodicity_is_template BOOLEAN DEFAULT FALSE;

-- Add indexes for periodicity queries
CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_at ON tasks (scheduled_at);
CREATE INDEX IF NOT EXISTS idx_tasks_periodicity_type ON tasks (periodicity_type);
CREATE INDEX IF NOT EXISTS idx_tasks_periodicity_next_execution ON tasks (periodicity_next_execution);
CREATE INDEX IF NOT EXISTS idx_tasks_periodicity_parent_task_id ON tasks (periodicity_parent_task_id);

-- Add foreign key constraint for parent task relationship
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_parent_task 
    FOREIGN KEY (periodicity_parent_task_id) REFERENCES tasks(id) ON DELETE SET NULL;
