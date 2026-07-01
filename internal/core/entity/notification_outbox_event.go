package entity

import (
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type NotificationOutboxEvent struct {
	DeleteableUUIDModel

	NotificationID uuid.UUID         `json:"notification_id" gorm:"type:uuid;not null;index"`
	Channel        enum.Channel      `json:"channel" gorm:"type:varchar(32);not null;index"`
	EventType      string            `json:"event_type" gorm:"type:varchar(128);not null;index"`
	Payload        types.JSONMap     `json:"payload,omitempty" gorm:"type:jsonb"`
	Status         enum.OutboxStatus `json:"status" gorm:"type:varchar(32);not null;index;index:idx_notification_outbox_events_due,priority:1"`
	Attempts       int               `json:"attempts" gorm:"not null;default:0"`
	NextAttemptAt  time.Time         `json:"next_attempt_at" gorm:"not null;index;index:idx_notification_outbox_events_due,priority:2"`
	LockedUntil    *time.Time        `json:"locked_until,omitempty" gorm:"index"`
	PublishedAt    *time.Time        `json:"published_at,omitempty" gorm:"index"`
	LastError      string            `json:"last_error,omitempty" gorm:"type:text"`
}
