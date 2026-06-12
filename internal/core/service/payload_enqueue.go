package service

import (
	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/event"
)

func enqueueFromPayload(payload event.SendNotificationPayload) command.Enqueue {
	return command.Enqueue{
		UserID:               payload.UserID,
		Channel:              payload.Channel,
		Recipients:           []string{payload.Recipient},
		Body:                 payload.Body,
		TemplateKey:          payload.TemplateKey,
		TemplateParams:       NormalizeJSONMap(payload.TemplateParams),
		ProviderTemplateID:   payload.ProviderTemplateID,
		ProviderTemplateVars: NormalizeJSONMap(payload.ProviderTemplateVars),
		Metadata:             NormalizeJSONMap(payload.Metadata),
	}
}
