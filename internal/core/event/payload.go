package event

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

// SendNotificationPayload is the queue message body for notification.send.
// One message maps to one notification row (single recipient).
type SendNotificationPayload struct {
	NotificationID uuid.UUID    `json:"notification_id,omitempty"`
	Channel        enum.Channel `json:"channel"`
	Recipient      string       `json:"recipient,omitempty"`
	CreatedAt      time.Time    `json:"created_at,omitempty"`

	UserID *uuid.UUID `json:"user_id,omitempty"`

	Body                 string        `json:"body,omitempty"`
	TemplateKey          string        `json:"template_key,omitempty"`
	TemplateParams       types.JSONMap `json:"template_params,omitempty"`
	ProviderTemplateID   string        `json:"provider_template_id,omitempty"`
	ProviderTemplateVars types.JSONMap `json:"provider_template_vars,omitempty"`
	Metadata             types.JSONMap `json:"metadata,omitempty"`
}

func (p SendNotificationPayload) HasNotificationID() bool {
	return p.NotificationID != uuid.Nil
}

// HasReferenceOnly is true for messages that only carry notification_id (row already in DB).
func (p SendNotificationPayload) HasReferenceOnly() bool {
	if !p.HasNotificationID() {
		return false
	}
	return strings.TrimSpace(p.Recipient) == "" &&
		strings.TrimSpace(p.Body) == "" &&
		strings.TrimSpace(p.TemplateKey) == "" &&
		strings.TrimSpace(p.ProviderTemplateID) == ""
}

// HasCreateFields is true when the payload carries enough data to build a new notification.
func (p SendNotificationPayload) HasCreateFields() bool {
	if strings.TrimSpace(p.Recipient) == "" {
		return false
	}
	if !p.Channel.IsValid() {
		return false
	}
	return p.Body != "" || p.TemplateKey != "" || p.ProviderTemplateID != ""
}
