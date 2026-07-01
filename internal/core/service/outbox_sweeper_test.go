package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/event"
)

func TestOutboxEventForNotificationIncludesServiceName(t *testing.T) {
	t.Parallel()

	notificationID := uuid.New()
	dueAt := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	notification := &entity.Notification{
		DeleteableUUIDModel: entity.DeleteableUUIDModel{ID: notificationID},
		ServiceName:         "billing-service",
		Channel:             enum.ChannelSMS,
		Recipient:           "+628111111111",
		Body:                "hello",
		Status:              enum.NotificationStatusQueued,
	}

	outboxEvent, err := OutboxEventForNotification(notification, dueAt)
	if err != nil {
		t.Fatalf("OutboxEventForNotification: %v", err)
	}
	if outboxEvent.NotificationID != notificationID {
		t.Fatalf("notification id = %v, want %v", outboxEvent.NotificationID, notificationID)
	}
	if outboxEvent.Status != enum.OutboxStatusPending {
		t.Fatalf("status = %v, want pending", outboxEvent.Status)
	}
	if !outboxEvent.NextAttemptAt.Equal(dueAt) {
		t.Fatalf("next attempt = %v, want %v", outboxEvent.NextAttemptAt, dueAt)
	}

	message, err := outboxMessageFromPayload(outboxEvent.Payload)
	if err != nil {
		t.Fatalf("outboxMessageFromPayload: %v", err)
	}
	if message.Payload.ServiceName != "billing-service" {
		t.Fatalf("service_name = %q, want billing-service", message.Payload.ServiceName)
	}
}

func TestOutboxSweeperPublishesAndMarksPublished(t *testing.T) {
	t.Parallel()

	row := newOutboxRowForTest(t, 1)
	outbox := &fakeOutboxStore{rows: []*entity.NotificationOutboxEvent{row}}
	publisher := &fakePublisher{}
	sweeper := NewOutboxSweeper(outbox, publisher, slog.Default(), OutboxSweeperConfig{
		BatchSize:      10,
		RetryBaseDelay: time.Second,
		RetryMaxDelay:  time.Minute,
	})

	count, err := sweeper.SweepOnce(context.Background())
	if err != nil {
		t.Fatalf("SweepOnce: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if len(publisher.published) != 1 {
		t.Fatalf("published calls = %d, want 1", len(publisher.published))
	}
	if len(outbox.publishedIDs) != 1 || outbox.publishedIDs[0] != row.ID {
		t.Fatalf("published ids = %v, want %v", outbox.publishedIDs, row.ID)
	}
	if len(outbox.pendingIDs) != 0 {
		t.Fatalf("pending ids = %v, want none", outbox.pendingIDs)
	}
}

func TestOutboxSweeperFailureMarksPendingAndContinues(t *testing.T) {
	t.Parallel()

	first := newOutboxRowForTest(t, 1)
	second := newOutboxRowForTest(t, 1)
	outbox := &fakeOutboxStore{rows: []*entity.NotificationOutboxEvent{first, second}}
	publisher := &fakePublisher{failFirst: true}
	sweeper := NewOutboxSweeper(outbox, publisher, slog.Default(), OutboxSweeperConfig{
		BatchSize:      10,
		RetryBaseDelay: time.Second,
		RetryMaxDelay:  time.Minute,
	})

	count, err := sweeper.SweepOnce(context.Background())
	if err != nil {
		t.Fatalf("SweepOnce: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if len(outbox.pendingIDs) != 1 || outbox.pendingIDs[0] != first.ID {
		t.Fatalf("pending ids = %v, want %v", outbox.pendingIDs, first.ID)
	}
	if len(outbox.publishedIDs) != 1 || outbox.publishedIDs[0] != second.ID {
		t.Fatalf("published ids = %v, want %v", outbox.publishedIDs, second.ID)
	}
	if !outbox.nextAttemptAt.After(time.Now()) {
		t.Fatalf("next attempt = %v, want future", outbox.nextAttemptAt)
	}
	if outbox.lastError != "publish boom" {
		t.Fatalf("last error = %q, want publish boom", outbox.lastError)
	}
}

func TestShouldSkipDispatch(t *testing.T) {
	t.Parallel()

	for _, status := range []enum.NotificationStatus{
		enum.NotificationStatusSent,
		enum.NotificationStatusDelivered,
		enum.NotificationStatusFailed,
	} {
		if !shouldSkipDispatch(status) {
			t.Fatalf("status %s should skip", status)
		}
	}
	for _, status := range []enum.NotificationStatus{
		enum.NotificationStatusQueued,
		enum.NotificationStatusRetrying,
		enum.NotificationStatusProcessing,
	} {
		if shouldSkipDispatch(status) {
			t.Fatalf("status %s should not skip", status)
		}
	}
}

func newOutboxRowForTest(t *testing.T, attempts int) *entity.NotificationOutboxEvent {
	t.Helper()

	message := event.OutboxMessage{
		ID:      uuid.New(),
		Type:    event.TypeNotificationSend,
		Channel: enum.ChannelSMS,
		Payload: event.SendNotificationPayload{
			NotificationID: uuid.New(),
			Channel:        enum.ChannelSMS,
			Recipient:      "+628111111111",
			ServiceName:    "billing-service",
			Body:           "hello",
		},
	}
	payload, err := outboxPayloadFromMessage(message)
	if err != nil {
		t.Fatalf("outboxPayloadFromMessage: %v", err)
	}
	return &entity.NotificationOutboxEvent{
		DeleteableUUIDModel: entity.DeleteableUUIDModel{ID: uuid.New()},
		NotificationID:      message.Payload.NotificationID,
		Channel:             message.Channel,
		EventType:           message.Type.String(),
		Payload:             payload,
		Status:              enum.OutboxStatusPublishing,
		Attempts:            attempts,
	}
}

type fakeOutboxStore struct {
	rows          []*entity.NotificationOutboxEvent
	publishedIDs  []uuid.UUID
	pendingIDs    []uuid.UUID
	nextAttemptAt time.Time
	lastError     string
}

func (f *fakeOutboxStore) ClaimDue(context.Context, time.Time, int, time.Duration) ([]*entity.NotificationOutboxEvent, error) {
	return f.rows, nil
}

func (f *fakeOutboxStore) MarkPublished(_ context.Context, id uuid.UUID, _ time.Time) error {
	f.publishedIDs = append(f.publishedIDs, id)
	return nil
}

func (f *fakeOutboxStore) MarkPending(_ context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error {
	f.pendingIDs = append(f.pendingIDs, id)
	f.nextAttemptAt = nextAttemptAt
	f.lastError = lastError
	return nil
}

type fakePublisher struct {
	failFirst bool
	calls     int
	published []event.OutboxMessage
}

func (f *fakePublisher) Publish(_ context.Context, message event.OutboxMessage) error {
	f.calls++
	if f.failFirst && f.calls == 1 {
		return errors.New("publish boom")
	}
	f.published = append(f.published, message)
	return nil
}
