package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/event"
)

type OutboxStore interface {
	ClaimDue(ctx context.Context, now time.Time, limit int, lockTTL time.Duration) ([]*entity.NotificationOutboxEvent, error)
	MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error
	MarkPending(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error
}

type OutboxSweeperConfig struct {
	SweepInterval  time.Duration
	BatchSize      int
	LockTTL        time.Duration
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

type OutboxSweeper struct {
	outbox    OutboxStore
	publisher event.Publisher
	log       *slog.Logger
	config    OutboxSweeperConfig
}

func NewOutboxSweeper(outbox OutboxStore, publisher event.Publisher, log *slog.Logger, config OutboxSweeperConfig) *OutboxSweeper {
	if config.SweepInterval <= 0 {
		config.SweepInterval = 2 * time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.LockTTL <= 0 {
		config.LockTTL = 30 * time.Second
	}
	if config.RetryBaseDelay <= 0 {
		config.RetryBaseDelay = 5 * time.Second
	}
	if config.RetryMaxDelay <= 0 {
		config.RetryMaxDelay = 5 * time.Minute
	}
	if log == nil {
		log = slog.Default()
	}
	return &OutboxSweeper{
		outbox:    outbox,
		publisher: publisher,
		log:       log,
		config:    config,
	}
}

func (s *OutboxSweeper) Run(ctx context.Context) error {
	if _, err := s.SweepOnce(ctx); err != nil && ctx.Err() == nil {
		s.log.ErrorContext(ctx, "outbox sweep failed", slog.String("error", err.Error()))
	}

	ticker := time.NewTicker(s.config.SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := s.SweepOnce(ctx); err != nil && ctx.Err() == nil {
				s.log.ErrorContext(ctx, "outbox sweep failed", slog.String("error", err.Error()))
			}
		}
	}
}

func (s *OutboxSweeper) SweepOnce(ctx context.Context) (int, error) {
	now := time.Now()
	rows, err := s.outbox.ClaimDue(ctx, now, s.config.BatchSize, s.config.LockTTL)
	if err != nil {
		return 0, err
	}

	for _, row := range rows {
		if err := s.publishRow(ctx, row); err != nil {
			s.log.WarnContext(ctx, "outbox publish failed",
				slog.String("outbox_id", row.ID.String()),
				slog.String("notification_id", row.NotificationID.String()),
				slog.String("error", err.Error()),
			)
		}
	}
	return len(rows), nil
}

func (s *OutboxSweeper) publishRow(ctx context.Context, row *entity.NotificationOutboxEvent) error {
	message, err := outboxMessageFromPayload(row.Payload)
	if err != nil {
		nextAttemptAt := time.Now().Add(s.retryDelay(row.Attempts))
		if markErr := s.outbox.MarkPending(ctx, row.ID, nextAttemptAt, truncateOutboxError(err.Error())); markErr != nil {
			return markErr
		}
		return err
	}

	if err := s.publisher.Publish(ctx, message); err != nil {
		nextAttemptAt := time.Now().Add(s.retryDelay(row.Attempts))
		if markErr := s.outbox.MarkPending(ctx, row.ID, nextAttemptAt, truncateOutboxError(err.Error())); markErr != nil {
			return markErr
		}
		return err
	}
	return s.outbox.MarkPublished(ctx, row.ID, time.Now())
}

func (s *OutboxSweeper) retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	delay := s.config.RetryBaseDelay
	for i := 1; i < attempts; i++ {
		if delay >= s.config.RetryMaxDelay {
			return s.config.RetryMaxDelay
		}
		if delay > s.config.RetryMaxDelay/2 {
			return s.config.RetryMaxDelay
		}
		delay *= 2
	}
	if delay > s.config.RetryMaxDelay {
		return s.config.RetryMaxDelay
	}
	return delay
}

func outboxMessageFromPayload(payload map[string]any) (event.OutboxMessage, error) {
	var message event.OutboxMessage
	raw, err := json.Marshal(payload)
	if err != nil {
		return message, err
	}
	if err := json.Unmarshal(raw, &message); err != nil {
		return message, err
	}
	return message, nil
}

func truncateOutboxError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 2048 {
		return value
	}
	return value[:2048]
}
