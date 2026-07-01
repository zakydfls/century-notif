package command

import (
	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

// Enqueue is the shared application payload for enqueue (REST batch or service/MQ mapping).
type Enqueue struct {
	UserID *uuid.UUID `json:"user_id,omitempty"`

	ServiceName string       `json:"-" validate:"omitempty,max=128"`
	Channel     enum.Channel `json:"channel" validate:"required,oneof=sms whatsapp email"`
	Recipients  []string     `json:"recipients,omitempty" validate:"omitempty,min=1,dive,required"`
	Body        string       `json:"body,omitempty"`

	TemplateKey          string        `json:"template_key,omitempty"`
	TemplateParams       types.JSONMap `json:"template_params,omitempty"`
	ProviderTemplateID   string        `json:"provider_template_id,omitempty"`
	ProviderTemplateVars types.JSONMap `json:"provider_template_vars,omitempty"`
	Metadata             types.JSONMap `json:"metadata,omitempty"`
}
type EnqueueSingle struct {
	Enqueue
	Recipient string `json:"recipient" validate:"required"`
}

func (s EnqueueSingle) ToEnqueue() Enqueue {
	e := s.Enqueue
	e.Recipients = []string{s.Recipient}
	return e
}

type EnqueueOneResult struct {
	NotificationID uuid.UUID `json:"notification_id"`
}

type EnqueueBatchResult struct {
	NotificationIDs []uuid.UUID `json:"notification_ids"`
}
