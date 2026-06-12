package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/event"
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
	notif     repository.Notification
	tx        *repository.TxManager
	publisher event.Publisher
	log       *slog.Logger
	builder   *NotificationBuilder
}

func NewNotificationService(
	notif repository.Notification,
	tx *repository.TxManager,
	publisher event.Publisher,
	log *slog.Logger,
	builder *NotificationBuilder,
) Notification {
	return &notificationService{notif: notif, tx: tx, publisher: publisher, log: log, builder: builder}
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
		return s.notif.WithTrx(tx).CreateMany(ctx, notifications)
	}); err != nil {
		return command.EnqueueBatchResult{}, err
	}

	s.publishToQueueAsync(ctx, notifications)

	ids := make([]uuid.UUID, 0, len(notifications))
	for _, notification := range notifications {
		ids = append(ids, notification.ID)
	}
	return command.EnqueueBatchResult{NotificationIDs: ids}, nil
}

func (s *notificationService) publishToQueueAsync(parent context.Context, notifications []*entity.Notification) {
	if len(notifications) == 0 {
		return
	}
	messages := make([]event.OutboxMessage, len(notifications))
	for i, n := range notifications {
		messages[i] = QueueMessageForNotification(n)
	}
	go func() {
		pubCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), 30*time.Second)
		defer cancel()
		for _, msg := range messages {
			if err := s.publisher.Publish(pubCtx, msg); err != nil {
				s.log.ErrorContext(pubCtx, "failed to publish notification to queue",
					slog.String("notification_id", msg.Payload.NotificationID.String()),
					slog.String("error", err.Error()),
				)
			}
		}
	}()
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
