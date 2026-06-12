-- +goose Up
CREATE TABLE IF NOT EXISTS notification_webhook_events (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    provider VARCHAR(32) NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    provider_message_id VARCHAR(255),
    notification_id UUID,
    status VARCHAR(32),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_created_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    error_message TEXT,
    payload JSONB,
    headers JSONB
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_webhook_event_provider_event_id
    ON notification_webhook_events(provider, event_id);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_provider ON notification_webhook_events(provider);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_event_type ON notification_webhook_events(event_type);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_provider_message_id ON notification_webhook_events(provider_message_id);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_notification_id ON notification_webhook_events(notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_status ON notification_webhook_events(status);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_received_at ON notification_webhook_events(received_at);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_event_created_at ON notification_webhook_events(event_created_at);
CREATE INDEX IF NOT EXISTS idx_notification_webhook_events_processed_at ON notification_webhook_events(processed_at);

-- +goose Down
DROP TABLE IF EXISTS notification_webhook_events;
