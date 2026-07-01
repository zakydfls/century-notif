package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/value_object/dto/pagination"
	apperror "notif-gateway/internal/value_object/handling/error"
)

// Notification is the application port for enqueue and read use cases.
type Notification interface {
	Enqueue(ctx context.Context, input command.Enqueue) (command.EnqueueOneResult, error)
	EnqueueBatch(ctx context.Context, input command.Enqueue) (command.EnqueueBatchResult, error)
	Find(ctx context.Context, e *entity.Notification) error
	FindAll(ctx context.Context, pg *pagination.Pagination) error
}

type notificationService struct {
	notif   repository.Notification
	outbox  repository.NotificationOutbox
	tx      txRunner
	builder *NotificationBuilder
}

type txRunner interface {
	Do(ctx context.Context, fn func(tx *gorm.DB) error) error
}

func NewNotificationService(
	notif repository.Notification,
	outbox repository.NotificationOutbox,
	tx txRunner,
	builder *NotificationBuilder,
) Notification {
	return &notificationService{notif: notif, outbox: outbox, tx: tx, builder: builder}
}

func (s *notificationService) Enqueue(ctx context.Context, input command.Enqueue) (command.EnqueueOneResult, error) {
	if len(input.Recipients) != 1 {
		return command.EnqueueOneResult{}, apperror.ErrBadRequest(nil, "enqueue requires exactly one recipient", apperror.KeyEnqueueSingleRecipient)
	}
	batch, err := s.EnqueueBatch(ctx, input)
	if err != nil {
		return command.EnqueueOneResult{}, err
	}
	if len(batch.NotificationIDs) == 0 {
		return command.EnqueueOneResult{}, nil
	}
	return command.EnqueueOneResult{NotificationID: batch.NotificationIDs[0]}, nil
}

func (s *notificationService) EnqueueBatch(ctx context.Context, input command.Enqueue) (command.EnqueueBatchResult, error) {
	notifications, err := s.builder.Build(ctx, input)
	if err != nil {
		return command.EnqueueBatchResult{}, err
	}

	if err := s.tx.Do(ctx, func(tx *gorm.DB) error {
		if err := s.notif.WithTrx(tx).CreateMany(ctx, notifications); err != nil {
			return err
		}

		now := time.Now()
		outboxEvents := make([]*entity.NotificationOutboxEvent, 0, len(notifications))
		for _, notification := range notifications {
			outboxEvent, err := OutboxEventForNotification(notification, now)
			if err != nil {
				return err
			}
			outboxEvents = append(outboxEvents, outboxEvent)
		}
		return s.outbox.WithTrx(tx).CreateMany(ctx, outboxEvents)
	}); err != nil {
		return command.EnqueueBatchResult{}, err
	}

	ids := make([]uuid.UUID, 0, len(notifications))
	for _, notification := range notifications {
		ids = append(ids, notification.ID)
	}
	return command.EnqueueBatchResult{NotificationIDs: ids}, nil
}

func (s *notificationService) Find(ctx context.Context, e *entity.Notification) error {
	if err := s.notif.Find(ctx, e); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.ErrNotFound(err, err.Error(), apperror.KeyRecordNotFound)
		}
		return err
	}
	return nil
}

func (s *notificationService) FindAll(ctx context.Context, pg *pagination.Pagination) error {
	return s.notif.FindAll(ctx, pg)
}
