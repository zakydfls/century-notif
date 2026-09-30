package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/value_object/dto/pagination"
)

func TestNotificationServiceEnqueueCreatesOutboxEventsWithServiceName(t *testing.T) {
	t.Parallel()

	notifRepo := &fakeNotificationRepo{}
	outboxRepo := &fakeNotificationOutboxRepo{}
	service := newNotificationServiceForTest(notifRepo, outboxRepo)

	result, err := service.EnqueueBatch(context.Background(), command.Enqueue{
		ServiceName: "billing-service",
		Channel:     enum.ChannelSMS,
		Recipients:  []string{"+628111111111", "+628122222222"},
		Body:        "hello",
	})
	if err != nil {
		t.Fatalf("EnqueueBatch: %v", err)
	}
	if len(result.NotificationIDs) != 2 {
		t.Fatalf("ids = %d, want 2", len(result.NotificationIDs))
	}
	if len(notifRepo.created) != 2 {
		t.Fatalf("notifications = %d, want 2", len(notifRepo.created))
	}
	if len(outboxRepo.created) != 2 {
		t.Fatalf("outbox events = %d, want 2", len(outboxRepo.created))
	}
	if notifRepo.created[0].ServiceName != "billing-service" {
		t.Fatalf("notification service_name = %q", notifRepo.created[0].ServiceName)
	}
	message, err := outboxMessageFromPayload(outboxRepo.created[0].Payload)
	if err != nil {
		t.Fatalf("outboxMessageFromPayload: %v", err)
	}
	if message.Payload.ServiceName != "billing-service" {
		t.Fatalf("payload service_name = %q", message.Payload.ServiceName)
	}
}

func TestNotificationServiceEnqueueReturnsOutboxError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("outbox unavailable")
	service := newNotificationServiceForTest(&fakeNotificationRepo{}, &fakeNotificationOutboxRepo{createErr: wantErr})

	_, err := service.EnqueueBatch(context.Background(), command.Enqueue{
		ServiceName: "billing-service",
		Channel:     enum.ChannelSMS,
		Recipients:  []string{"+628111111111"},
		Body:        "hello",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func newNotificationServiceForTest(notifRepo repository.Notification, outboxRepo repository.NotificationOutbox) Notification {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dispatcher := NewDispatcher(DispatcherConfig{
		DefaultSMS:      enum.ProviderTwilio,
		DefaultWhatsApp: enum.ProviderTwilio,
		DefaultEmail:    enum.ProviderGoDaddy,
	}, nil, logger)
	return NewNotificationService(notifRepo, outboxRepo, fakeTxRunner{}, NewNotificationBuilder(nil, dispatcher, EmailSecurityPolicy{}))
}

type fakeTxRunner struct{}

func (fakeTxRunner) Do(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return fn(nil)
}

type fakeNotificationRepo struct {
	created []*entity.Notification
}

func (f *fakeNotificationRepo) WithTrx(*gorm.DB) repository.Notification { return f }
func (f *fakeNotificationRepo) Custom(context.Context, func(db *gorm.DB) *gorm.DB) error {
	return nil
}
func (f *fakeNotificationRepo) CustomError(context.Context, func(db *gorm.DB) error) error {
	return nil
}
func (f *fakeNotificationRepo) Create(_ context.Context, e *entity.Notification) error {
	f.created = append(f.created, e)
	return nil
}
func (f *fakeNotificationRepo) CreateMany(_ context.Context, es []*entity.Notification) error {
	for _, e := range es {
		_ = e.BeforeCreate(nil)
		f.created = append(f.created, e)
	}
	return nil
}
func (f *fakeNotificationRepo) Find(context.Context, *entity.Notification) error {
	return repository.ErrNotFound
}
func (f *fakeNotificationRepo) Save(context.Context, *entity.Notification) error { return nil }
func (f *fakeNotificationRepo) FindAll(context.Context, *pagination.Pagination) error {
	return nil
}

type fakeNotificationOutboxRepo struct {
	created   []*entity.NotificationOutboxEvent
	createErr error
}

func (f *fakeNotificationOutboxRepo) WithTrx(*gorm.DB) repository.NotificationOutbox { return f }
func (f *fakeNotificationOutboxRepo) Custom(context.Context, func(db *gorm.DB) *gorm.DB) error {
	return nil
}
func (f *fakeNotificationOutboxRepo) CustomError(context.Context, func(db *gorm.DB) error) error {
	return nil
}
func (f *fakeNotificationOutboxRepo) CreateMany(_ context.Context, es []*entity.NotificationOutboxEvent) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, es...)
	return nil
}
func (f *fakeNotificationOutboxRepo) ClaimDue(context.Context, time.Time, int, time.Duration) ([]*entity.NotificationOutboxEvent, error) {
	return nil, nil
}
func (f *fakeNotificationOutboxRepo) MarkPublished(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (f *fakeNotificationOutboxRepo) MarkPending(context.Context, uuid.UUID, time.Time, string) error {
	return nil
}
