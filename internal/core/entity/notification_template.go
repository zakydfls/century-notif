package entity

import (
	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type NotificationTemplate struct {
	DeleteableUUIDModel

	Key                string        `json:"key" gorm:"type:varchar(128);not null;index:idx_notification_template_lookup,priority:1"`
	Channel            enum.Channel  `json:"channel" gorm:"type:varchar(32);not null;index:idx_notification_template_lookup,priority:2"`
	Provider           enum.Provider `json:"provider" gorm:"type:varchar(32);not null;index"`
	Body               string        `json:"body" gorm:"type:text;not null"`
	ProviderTemplateID string        `json:"provider_template_id,omitempty" gorm:"type:varchar(128)"`
	Metadata           types.JSONMap `json:"metadata,omitempty" gorm:"type:jsonb"`
	Active             bool          `json:"active" gorm:"not null;default:true;index"`
}
