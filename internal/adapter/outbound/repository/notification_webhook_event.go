package repository

import (
	"context"

	"gorm.io/gorm"

	"notif-gateway/internal/core/entity"
)

type WebhookEvent interface {
	BaseRepo[WebhookEvent]

	Create(ctx context.Context, e *entity.NotificationWebhookEvent) error
	Save(ctx context.Context, e *entity.NotificationWebhookEvent) error
}

func NewWebhookEvent(db *gorm.DB) WebhookEvent {
	fac := func(base *baseRepo[WebhookEvent]) WebhookEvent {
		return &webhookEvent{baseRepo: base}
	}
	return newRepository(fac, db)
}

type webhookEvent struct {
	*baseRepo[WebhookEvent]
}

func (r *webhookEvent) Create(ctx context.Context, e *entity.NotificationWebhookEvent) error {
	if err := r.dbCtx(ctx).Create(e).Error; err != nil {
		if IsDuplicateError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *webhookEvent) Save(ctx context.Context, e *entity.NotificationWebhookEvent) error {
	return r.dbCtx(ctx).Save(e).Error
}
