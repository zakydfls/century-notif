package service

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/event"
	types "notif-gateway/internal/value_object/util/type"
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
		ServiceName:          n.ServiceName,
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

func OutboxEventForNotification(notification *entity.Notification, now time.Time) (*entity.NotificationOutboxEvent, error) {
	message := QueueMessageForNotification(notification)
	payload, err := outboxPayloadFromMessage(message)
	if err != nil {
		return nil, err
	}
	return &entity.NotificationOutboxEvent{
		NotificationID: notification.ID,
		Channel:        notification.Channel,
		EventType:      message.Type.String(),
		Payload:        payload,
		Status:         enum.OutboxStatusPending,
		NextAttemptAt:  now,
	}, nil
}

func outboxPayloadFromMessage(message event.OutboxMessage) (types.JSONMap, error) {
	raw, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return types.NewJSONMap(payload), nil
}
