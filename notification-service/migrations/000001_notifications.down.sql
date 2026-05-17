DROP INDEX IF EXISTS idx_notification_events_type_created;
DROP INDEX IF EXISTS idx_notifications_recipient_created;
DROP INDEX IF EXISTS idx_notifications_recipient_read_created;

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS notification_events;
