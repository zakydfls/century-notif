package entity

import (
	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type NotificationAttempt struct {
	DeleteableUUIDModel

	NotificationID  uuid.UUID               `json:"notification_id" gorm:"type:uuid;not null;index"`
	Channel         enum.Channel            `json:"channel" gorm:"type:varchar(32);not null;index"`
	Provider        enum.Provider           `json:"provider" gorm:"type:varchar(32);not null;index"`
	Status          enum.NotificationStatus `json:"status" gorm:"type:varchar(32);not null;index"`
	ErrorMessage    string                  `json:"error_message,omitempty" gorm:"type:text"`
	RequestPayload  types.JSONMap           `json:"request_payload,omitempty" gorm:"type:jsonb"`
	ResponsePayload types.JSONMap           `json:"response_payload,omitempty" gorm:"type:jsonb"`
}
