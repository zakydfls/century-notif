package service

import (
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/event"
)

func payloadFromNotification(n *entity.Notification) event.SendNotificationPayload {
	createdAt := time.Now()
	if !n.CreatedAt.IsZero() {
		createdAt = n.CreatedAt
	}
	return event.SendNotificationPayload{
		NotificationID:       n.ID,
		Channel:              n.Channel,
		Recipient:            n.Recipient,
		CreatedAt:            createdAt,
		UserID:               n.UserID,
		Body:                 n.Body,
		TemplateKey:          n.TemplateKey,
		TemplateParams:       n.TemplateParams,
		ProviderTemplateID:   n.ProviderTemplateID,
		ProviderTemplateVars: n.ProviderTemplateVars,
		Metadata:             n.Metadata,
	}
}

func QueueMessageForNotification(notification *entity.Notification) event.OutboxMessage {
	return event.OutboxMessage{
		ID:      uuid.New(),
		Type:    event.TypeNotificationSend,
		Channel: notification.Channel,
		Payload: payloadFromNotification(notification),
	}
}
