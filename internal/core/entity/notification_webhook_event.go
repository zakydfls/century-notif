package entity

import (
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type NotificationWebhookEvent struct {
	DeleteableUUIDModel

	Provider          enum.Provider           `json:"provider" gorm:"type:varchar(32);not null;index;uniqueIndex:idx_webhook_event_provider_event_id"`
	EventID           string                  `json:"event_id" gorm:"type:varchar(255);not null;uniqueIndex:idx_webhook_event_provider_event_id"`
	EventType         string                  `json:"event_type" gorm:"type:varchar(128);not null;index"`
	ProviderMessageID string                  `json:"provider_message_id,omitempty" gorm:"type:varchar(255);index"`
	NotificationID    *uuid.UUID              `json:"notification_id,omitempty" gorm:"type:uuid;index"`
	Status            enum.NotificationStatus `json:"status,omitempty" gorm:"type:varchar(32);index"`
	ReceivedAt        time.Time               `json:"received_at" gorm:"not null;index"`
	EventCreatedAt    *time.Time              `json:"event_created_at,omitempty" gorm:"index"`
	ProcessedAt       *time.Time              `json:"processed_at,omitempty" gorm:"index"`
	ErrorMessage      string                  `json:"error_message,omitempty" gorm:"type:text"`
	Payload           types.JSONMap           `json:"payload,omitempty" gorm:"type:jsonb"`
	Headers           types.JSONMap           `json:"headers,omitempty" gorm:"type:jsonb"`
}
