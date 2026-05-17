CREATE TABLE IF NOT EXISTS notification_tasks (
    task_id UUID PRIMARY KEY,
    project_id UUID,
    assignee_user_id UUID,
    task_title VARCHAR(255),
    project_name VARCHAR(255),
    deadline TIMESTAMP WITH TIME ZONE,
    status INTEGER NOT NULL DEFAULT 0,
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notification_tasks_assignee_deadline
    ON notification_tasks (assignee_user_id, deadline);

CREATE INDEX IF NOT EXISTS idx_notification_tasks_deadline_status
    ON notification_tasks (deadline, status);
