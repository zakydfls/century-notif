package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/value_object/dto/pagination"
	apperror "notif-gateway/internal/value_object/handling/error"
)

func TestNotificationKeyMiddlewareSetsServiceNameFromHeader(t *testing.T) {
	t.Parallel()

	handler := notificationKeyMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := serviceNameFromContext(r.Context()); got != "billing-service" {
			t.Fatalf("service name = %q, want billing-service", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", nil)
	req.Header.Set(enum.HeaderNotificationKey, "secret")
	req.Header.Set(enum.HeaderCenturyService, " billing-service ")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestNotificationKeyMiddlewareDefaultsMissingServiceName(t *testing.T) {
	t.Parallel()

	handler := notificationKeyMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := serviceNameFromContext(r.Context()); got != unknownServiceName {
			t.Fatalf("service name = %q, want %s", got, unknownServiceName)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", nil)
	req.Header.Set(enum.HeaderNotificationKey, "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestNotificationRESTEnqueueUsesContextServiceName(t *testing.T) {
	t.Parallel()

	notifications := &captureNotificationService{}
	rest := &notificationREST{notifications: notifications}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", strings.NewReader(`{
		"channel": "sms",
		"recipient": "+628111111111",
		"body": "hello"
	}`))
	req = req.WithContext(context.WithValue(req.Context(), serviceNameContextKey{}, "billing-service"))
	rec := httptest.NewRecorder()

	if err := rest.Enqueue(rec, req); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if notifications.single.ServiceName != "billing-service" {
		t.Fatalf("service name = %q, want billing-service", notifications.single.ServiceName)
	}
}

func TestNotificationRESTRejectsServiceNameInBody(t *testing.T) {
	t.Parallel()

	notifications := &captureNotificationService{}
	rest := &notificationREST{notifications: notifications}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", strings.NewReader(`{
		"service_name": "billing-service",
		"channel": "sms",
		"recipient": "+628111111111",
		"body": "hello"
	}`))
	req = req.WithContext(context.WithValue(req.Context(), serviceNameContextKey{}, "header-service"))
	rec := httptest.NewRecorder()

	if err := rest.Enqueue(rec, req); err == nil {
		t.Fatal("Enqueue nil error, want unknown service_name body field error")
	}
	if notifications.single.Channel != "" {
		t.Fatalf("notification service was called: %+v", notifications.single)
	}
}

func TestNotificationRESTReturnsDisposableEmailErrorContract(t *testing.T) {
	t.Parallel()

	notifications := &captureNotificationService{enqueueErr: apperror.ErrBadRequest(nil,
		"disposable email addresses are not allowed", apperror.KeyEmailDisposableRecipient)}
	rest := &notificationREST{notifications: notifications}
	handler := requestIDMiddleware(RootHandler(rest.Enqueue))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications", strings.NewReader(`{
		"channel":"email",
		"recipient":"user@mailinator.com",
		"body":"123456",
		"metadata":{"subject":"OTP"}
	}`))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		`"code":"BAD_REQUEST"`,
		`"key":"notification.email.disposable_recipient"`,
		`"message":"disposable email addresses are not allowed"`,
		`"request_id":`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body %s does not contain %s", body, expected)
		}
	}
}

type captureNotificationService struct {
	single     command.Enqueue
	batch      command.Enqueue
	enqueueErr error
}

func (c *captureNotificationService) Enqueue(_ context.Context, input command.Enqueue) (command.EnqueueOneResult, error) {
	c.single = input
	if c.enqueueErr != nil {
		return command.EnqueueOneResult{}, c.enqueueErr
	}
	return command.EnqueueOneResult{NotificationID: uuid.New()}, nil
}

func (c *captureNotificationService) EnqueueBatch(_ context.Context, input command.Enqueue) (command.EnqueueBatchResult, error) {
	c.batch = input
	return command.EnqueueBatchResult{NotificationIDs: []uuid.UUID{uuid.New()}}, nil
}

func (c *captureNotificationService) Find(context.Context, *entity.Notification) error {
	return nil
}

func (c *captureNotificationService) FindAll(context.Context, *pagination.Pagination) error {
	return nil
}
