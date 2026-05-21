ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS activity_type_id UUID REFERENCES activity_types(id),
    ADD COLUMN IF NOT EXISTS planned_hours NUMERIC(10, 2),
    ADD COLUMN IF NOT EXISTS actual_hours NUMERIC(10, 2);

CREATE INDEX IF NOT EXISTS idx_tasks_activity_type ON tasks(activity_type_id);
