CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS notification_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    event_key VARCHAR(128) NOT NULL UNIQUE,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipient_user_id UUID NOT NULL,
    type VARCHAR(100) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 1,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    project_id UUID,
    task_id UUID,
    actor_user_id UUID,
    action_url VARCHAR(500) NOT NULL DEFAULT '',
    source_event_type VARCHAR(100) NOT NULL,
    source_event_key VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(recipient_user_id, source_event_key)
);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient_read_created
    ON notifications (recipient_user_id, read_at, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient_created
    ON notifications (recipient_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notification_events_type_created
    ON notification_events (event_type, created_at DESC);
