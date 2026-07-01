package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
)

type NotificationOutbox interface {
	BaseRepo[NotificationOutbox]

	CreateMany(ctx context.Context, es []*entity.NotificationOutboxEvent) error
	ClaimDue(ctx context.Context, now time.Time, limit int, lockTTL time.Duration) ([]*entity.NotificationOutboxEvent, error)
	MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error
	MarkPending(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error
}

func NewNotificationOutbox(db *gorm.DB) NotificationOutbox {
	fac := func(base *baseRepo[NotificationOutbox]) NotificationOutbox {
		return &notificationOutbox{baseRepo: base}
	}
	return newRepository(fac, db)
}

type notificationOutbox struct {
	*baseRepo[NotificationOutbox]
}

func (r *notificationOutbox) CreateMany(ctx context.Context, es []*entity.NotificationOutboxEvent) error {
	if len(es) == 0 {
		return nil
	}
	return r.dbCtx(ctx).Create(&es).Error
}

func (r *notificationOutbox) ClaimDue(ctx context.Context, now time.Time, limit int, lockTTL time.Duration) ([]*entity.NotificationOutboxEvent, error) {
	if limit < 1 {
		limit = 1
	}
	if lockTTL <= 0 {
		lockTTL = 30 * time.Second
	}

	lockedUntil := now.Add(lockTTL)
	var rows []*entity.NotificationOutboxEvent
	err := r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(status = ? AND next_attempt_at <= ?) OR (status = ? AND locked_until <= ?)",
				enum.OutboxStatusPending, now,
				enum.OutboxStatusPublishing, now,
			).
			Order("next_attempt_at ASC, created_at ASC").
			Limit(limit).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		ids := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		if err := tx.Model(&entity.NotificationOutboxEvent{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":       enum.OutboxStatusPublishing,
				"attempts":     gorm.Expr("attempts + 1"),
				"locked_until": lockedUntil,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}
		return tx.
			Where("id IN ?", ids).
			Order("next_attempt_at ASC, created_at ASC").
			Find(&rows).Error
	})
	return rows, err
}

func (r *notificationOutbox) MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error {
	return r.dbCtx(ctx).
		Model(&entity.NotificationOutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       enum.OutboxStatusPublished,
			"published_at": publishedAt,
			"locked_until": nil,
			"last_error":   "",
			"updated_at":   publishedAt,
		}).Error
}

func (r *notificationOutbox) MarkPending(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error {
	now := time.Now()
	return r.dbCtx(ctx).
		Model(&entity.NotificationOutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":          enum.OutboxStatusPending,
			"next_attempt_at": nextAttemptAt,
			"locked_until":    nil,
			"last_error":      lastError,
			"updated_at":      now,
		}).Error
}
