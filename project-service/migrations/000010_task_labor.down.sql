DROP INDEX IF EXISTS idx_tasks_activity_type;
ALTER TABLE tasks
    DROP COLUMN IF EXISTS actual_hours,
    DROP COLUMN IF EXISTS planned_hours,
    DROP COLUMN IF EXISTS activity_type_id;
