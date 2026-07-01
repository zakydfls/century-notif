package entity

import (
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type Notification struct {
	DeleteableUUIDModel

	UserID *uuid.UUID `json:"user_id,omitempty" gorm:"index"`

	ServiceName string        `json:"service_name,omitempty" gorm:"type:varchar(128);not null;default:'';index"`
	Channel     enum.Channel  `json:"channel" gorm:"type:varchar(32);not null;index"`
	Provider    enum.Provider `json:"provider" gorm:"type:varchar(32);not null;index"`
	Recipient   string        `json:"recipient" gorm:"type:varchar(255);not null;index"`

	Body                 string        `json:"body" gorm:"type:text"`
	TemplateKey          string        `json:"template_key,omitempty" gorm:"type:varchar(128);index"`
	TemplateParams       types.JSONMap `json:"template_params,omitempty" gorm:"type:jsonb"`
	ProviderTemplateID   string        `json:"provider_template_id,omitempty" gorm:"type:varchar(128)"`
	ProviderTemplateVars types.JSONMap `json:"provider_template_vars,omitempty" gorm:"type:jsonb"`
	Metadata             types.JSONMap `json:"metadata,omitempty" gorm:"type:jsonb"`

	Status    enum.NotificationStatus `json:"status" gorm:"type:varchar(32);not null;index"`
	Attempts  int                     `json:"attempts" gorm:"not null;default:0"`
	LastError string                  `json:"last_error,omitempty" gorm:"type:text"`
	// ProviderMessageID stores the provider message id after send (e.g. Twilio MessageSid). Twilio status webhooks
	// typically only send MessageSid; WebhookHandler resolves the row via Find(Provider, ProviderMessageID) when notification_id is absent.
	ProviderMessageID string `json:"provider_message_id,omitempty" gorm:"type:varchar(255);index"`

	SentAt      *time.Time `json:"sent_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
}
