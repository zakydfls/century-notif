package command

import (
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type HandleWebhook struct {
	Provider             enum.Provider
	NotificationID       uuid.UUID
	ProviderMessageID    string
	ExternalStatus       string
	ErrorMessage         string
	ErrorCode            string
	ChannelStatusMessage string // Twilio channel-level failure detail (WhatsApp, RCS)
}

// WebhookInbound is the JSON body for generic provider webhooks (Alibaba, WAHA, etc.).
type WebhookInbound struct {
	NotificationID    uuid.UUID `json:"notification_id,omitempty"`
	ProviderMessageID string    `json:"provider_message_id,omitempty"`
	Status            string    `json:"status" validate:"required"`
	ErrorMessage      string    `json:"error_message,omitempty"`
}

func (w WebhookInbound) ToHandleWebhook(provider enum.Provider) HandleWebhook {
	return HandleWebhook{
		Provider:          provider,
		NotificationID:    w.NotificationID,
		ProviderMessageID: w.ProviderMessageID,
		ExternalStatus:    w.Status,
		ErrorMessage:      w.ErrorMessage,
	}
}

type HandleResendWebhook struct {
	EventID           string
	EventType         string
	EventCreatedAt    time.Time
	ProviderMessageID string
	Data              types.JSONMap
	Payload           types.JSONMap
	Headers           types.JSONMap
}

type HandleWAHAWebhook struct {
	EventID           string
	EventType         string
	EventCreatedAt    time.Time
	Session           string
	ProviderMessageID string
	AckName           string
	Ack               int
	AckPresent        bool
	Payload           types.JSONMap
	Data              types.JSONMap
	Headers           types.JSONMap
}
