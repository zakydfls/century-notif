package service

import (
	"io"
	"log/slog"
	"testing"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

func TestNotificationBuilderDisposableEmailPolicy(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher(DispatcherConfig{DefaultEmail: enum.ProviderResend}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	input := command.Enqueue{
		Channel:    enum.ChannelEmail,
		Recipients: []string{"user@mailinator.com"},
		Body:       "123456",
		Metadata:   types.JSONMap{enum.EmailMetaSubject: "OTP"},
	}

	builder := NewNotificationBuilder(nil, dispatcher, EmailSecurityPolicy{BlockDisposable: true})
	if _, err := builder.Build(t.Context(), input); !isAppErrorKey(err, apperror.KeyEmailDisposableRecipient) {
		t.Fatalf("error = %v, want %s", err, apperror.KeyEmailDisposableRecipient)
	}

	input.Recipients = []string{"user@sub.mailinator.com"}
	if _, err := builder.Build(t.Context(), input); !isAppErrorKey(err, apperror.KeyEmailDisposableRecipient) {
		t.Fatalf("subdomain error = %v, want %s", err, apperror.KeyEmailDisposableRecipient)
	}

	builder = NewNotificationBuilder(nil, dispatcher, EmailSecurityPolicy{
		BlockDisposable: true, DisposableAllowlist: []string{"MAILINATOR.COM"},
	})
	if _, err := builder.Build(t.Context(), input); err != nil {
		t.Fatalf("allowlisted recipient rejected: %v", err)
	}

	input.Recipients = []string{"user@gmail.com"}
	if _, err := builder.Build(t.Context(), input); err != nil {
		t.Fatalf("regular recipient rejected: %v", err)
	}

	builder = NewNotificationBuilder(nil, dispatcher, EmailSecurityPolicy{BlockDisposable: true})
	input.Metadata[enum.EmailMetaBCC] = []any{"user@mailinator.com"}
	if _, err := builder.Build(t.Context(), input); !isAppErrorKey(err, apperror.KeyEmailDisposableRecipient) {
		t.Fatalf("bcc error = %v, want %s", err, apperror.KeyEmailDisposableRecipient)
	}
}

func isAppErrorKey(err error, key string) bool {
	appErr, ok := err.(*apperror.Error)
	return ok && appErr.Key == key
}
