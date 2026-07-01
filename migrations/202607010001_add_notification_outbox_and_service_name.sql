-- +goose Up
ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS service_name VARCHAR(128) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_notifications_service_name ON notifications(service_name);

CREATE TABLE IF NOT EXISTS notification_outbox_events (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notification_id UUID NOT NULL,
    channel VARCHAR(32) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB,
    status VARCHAR(32) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    locked_until TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_notification_outbox_events_notification_id
    ON notification_outbox_events(notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_events_channel
    ON notification_outbox_events(channel);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_events_event_type
    ON notification_outbox_events(event_type);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_events_status
    ON notification_outbox_events(status);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_events_due
    ON notification_outbox_events(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_events_locked_until
    ON notification_outbox_events(locked_until);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_events_published_at
    ON notification_outbox_events(published_at);

-- +goose Down
DROP TABLE IF EXISTS notification_outbox_events;
DROP INDEX IF EXISTS idx_notifications_service_name;
ALTER TABLE notifications DROP COLUMN IF EXISTS service_name;
