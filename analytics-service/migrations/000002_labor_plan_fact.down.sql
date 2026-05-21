DROP INDEX IF EXISTS idx_task_analytics_activity;
ALTER TABLE task_analytics
    DROP COLUMN IF EXISTS actual_hours,
    DROP COLUMN IF EXISTS planned_hours,
    DROP COLUMN IF EXISTS activity_type_id;
DROP TABLE IF EXISTS activity_types;
