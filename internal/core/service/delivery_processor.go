package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/event"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

// Delivery processes a notification.send message from the queue.
type Delivery interface {
	Process(ctx context.Context, message event.OutboxMessage) error
}

var _ Delivery = (*deliveryProcessor)(nil)

type deliveryProcessor struct {
	notif      repository.Notification
	attempt    repository.Attempt
	dispatcher *Dispatcher
	builder    *NotificationBuilder
}

func NewDeliveryProcessor(
	notif repository.Notification,
	attempt repository.Attempt,
	dispatcher *Dispatcher,
	builder *NotificationBuilder,
) Delivery {
	return &deliveryProcessor{
		notif:      notif,
		attempt:    attempt,
		dispatcher: dispatcher,
		builder:    builder,
	}
}

func (p *deliveryProcessor) Process(ctx context.Context, message event.OutboxMessage) error {
	notification, err := p.resolveNotification(ctx, message.Payload)
	if err != nil {
		return err
	}

	notification.Status = enum.NotificationStatusProcessing
	if err := p.notif.Save(ctx, notification); err != nil {
		return err
	}
	result, err := p.dispatcher.Dispatch(ctx, command.Dispatch{
		NotificationID:       notification.ID.String(),
		Channel:              notification.Channel,
		Recipient:            notification.Recipient,
		Body:                 notification.Body,
		ProviderTemplateID:   notification.ProviderTemplateID,
		ProviderTemplateVars: notification.ProviderTemplateVars,
		Metadata:             notification.Metadata,
	})
	if err != nil {
		p.markFailedAttempt(ctx, notification, err)
		return err
	}
	p.markSendSuccess(ctx, notification, result)
	return nil
}

func (p *deliveryProcessor) resolveNotification(ctx context.Context, payload event.SendNotificationPayload) (*entity.Notification, error) {
	if payload.HasNotificationID() {
		existing := &entity.Notification{}
		existing.ID = payload.NotificationID
		err := p.notif.Find(ctx, existing)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		return p.createNotificationFromPayload(ctx, payload)
	}

	if !payload.HasCreateFields() {
		return nil, apperror.ErrBadRequest(nil, "body, template key, or provider template id is required", apperror.KeyMissingContent)
	}
	return p.createNotificationFromPayload(ctx, payload)
}

func (p *deliveryProcessor) createNotificationFromPayload(ctx context.Context, payload event.SendNotificationPayload) (*entity.Notification, error) {
	if !payload.HasCreateFields() {
		return nil, apperror.ErrBadRequest(nil, "body, template key, or provider template id is required", apperror.KeyMissingContent)
	}

	notifications, err := p.builder.Build(ctx, enqueueFromPayload(payload))
	if err != nil {
		return nil, err
	}
	if len(notifications) != 1 {
		return nil, apperror.ErrBadRequest(nil, "expected exactly one notification from payload", apperror.KeyPayloadSingleNotification)
	}
	notification := notifications[0]
	if payload.HasNotificationID() {
		notification.ID = payload.NotificationID
	}

	if err := p.notif.Create(ctx, notification); err != nil {
		if payload.HasNotificationID() && isDuplicateKeyError(err) {
			existing := &entity.Notification{}
			existing.ID = payload.NotificationID
			if findErr := p.notif.Find(ctx, existing); findErr == nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return notification, nil
}

func isDuplicateKeyError(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

func (p *deliveryProcessor) markFailedAttempt(ctx context.Context, notification *entity.Notification, dispatchErr error) {
	notification.Attempts++
	notification.LastError = dispatchErr.Error()
	notification.Status = enum.NotificationStatusRetrying
	_ = p.notif.Save(ctx, notification)
	_ = p.attempt.Create(ctx, &entity.NotificationAttempt{
		NotificationID:  notification.ID,
		Channel:         notification.Channel,
		Provider:        notification.Provider,
		Status:          enum.NotificationStatusFailed,
		ErrorMessage:    dispatchErr.Error(),
		RequestPayload:  types.JSONMap{"notification_id": notification.ID.String()},
		ResponsePayload: types.JSONMap{},
	})
}

func (p *deliveryProcessor) markSendSuccess(ctx context.Context, notification *entity.Notification, result command.ProviderSendResult) {
	now := time.Now()
	status := p.dispatcher.StatusAfterSuccessfulSend(notification.Channel, notification.Provider)

	notification.Attempts++
	notification.Status = status
	notification.ProviderMessageID = result.MessageID
	notification.LastError = ""
	notification.SentAt = &now
	if !p.dispatcher.SupportsDeliveryWebhook(notification.Channel, notification.Provider) {
		notification.DeliveredAt = &now
	}

	_ = p.notif.Save(ctx, notification)
	_ = p.attempt.Create(ctx, &entity.NotificationAttempt{
		NotificationID:  notification.ID,
		Channel:         notification.Channel,
		Provider:        notification.Provider,
		Status:          status,
		RequestPayload:  types.JSONMap{"notification_id": notification.ID.String()},
		ResponsePayload: NormalizeJSONMap(result.Raw),
	})
}
