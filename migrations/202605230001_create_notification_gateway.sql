-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID,
    channel VARCHAR(32) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    body TEXT,
    template_key VARCHAR(128),
    template_params JSONB,
    provider_template_id VARCHAR(128),
    provider_template_vars JSONB,
    metadata JSONB,
    status VARCHAR(32) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    provider_message_id VARCHAR(255),
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_channel ON notifications(channel);
CREATE INDEX IF NOT EXISTS idx_notifications_provider ON notifications(provider);
CREATE INDEX IF NOT EXISTS idx_notifications_recipient ON notifications(recipient);
CREATE INDEX IF NOT EXISTS idx_notifications_template_key ON notifications(template_key);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_provider_message_id ON notifications(provider_message_id);

CREATE TABLE IF NOT EXISTS notification_attempts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notification_id UUID NOT NULL,
    channel VARCHAR(32) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    error_message TEXT,
    request_payload JSONB,
    response_payload JSONB
);

CREATE INDEX IF NOT EXISTS idx_notification_attempts_notification_id ON notification_attempts(notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_attempts_channel ON notification_attempts(channel);
CREATE INDEX IF NOT EXISTS idx_notification_attempts_provider ON notification_attempts(provider);
CREATE INDEX IF NOT EXISTS idx_notification_attempts_status ON notification_attempts(status);

CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    key VARCHAR(128) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    body TEXT NOT NULL,
    provider_template_id VARCHAR(128),
    metadata JSONB,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_notification_templates_provider ON notification_templates(provider);
CREATE INDEX IF NOT EXISTS idx_notification_templates_active ON notification_templates(active);
CREATE INDEX IF NOT EXISTS idx_notification_template_lookup ON notification_templates(key, channel);

-- +goose Down
DROP TABLE IF EXISTS notification_templates;
DROP TABLE IF EXISTS notification_attempts;
DROP TABLE IF EXISTS notifications;
