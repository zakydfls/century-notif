package service

import (
	"testing"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

func TestMapProviderStatus_TwilioAndAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want enum.NotificationStatus
	}{
		{"Delivered", enum.NotificationStatusDelivered},
		{enum.TwilioMessageStatusRead, enum.NotificationStatusDelivered},
		{enum.TwilioMessageStatusFailed, enum.NotificationStatusFailed},
		{enum.TwilioMessageStatusUndelivered, enum.NotificationStatusFailed},
		{enum.TwilioMessageStatusCancelled, enum.NotificationStatusFailed},
		{enum.TwilioMessageStatusCanceled, enum.NotificationStatusFailed},
		{enum.TwilioMessageStatusQueued, enum.NotificationStatusQueued},
		{enum.TwilioMessageStatusScheduled, enum.NotificationStatusQueued},
		{enum.TwilioMessageStatusAccepted, enum.NotificationStatusQueued},
		{enum.TwilioMessageStatusSent, enum.NotificationStatusSent},
		{enum.TwilioMessageStatusSending, enum.NotificationStatusProcessing},
		{enum.TwilioMessageStatusReceiving, enum.NotificationStatusProcessing},
		{enum.WebhookStatusSuccess, enum.NotificationStatusDelivered},
		{enum.WebhookStatusRejected, enum.NotificationStatusFailed},
		{"  ", enum.NotificationStatusProcessing},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := MapProviderStatus(tt.in)
			if got != tt.want {
				t.Fatalf("MapProviderStatus(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatTwilioWebhookError(t *testing.T) {
	t.Parallel()

	if got := formatTwilioWebhookError("30007", "Carrier violation", ""); got != "30007: Carrier violation" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := formatTwilioWebhookError("", "oops", ""); got != "oops" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := formatTwilioWebhookError("1", "x", "x"); got != "1: x" {
		t.Fatalf("dedupe channel same as msg: want %q, got %q", "1: x", got)
	}
	if got := formatTwilioWebhookError("2", "", "detail"); got != "2: detail" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestMapResendEventStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in     string
		want   enum.NotificationStatus
		wantOK bool
	}{
		{enum.ResendEventEmailScheduled, enum.NotificationStatusQueued, true},
		{enum.ResendEventEmailSent, enum.NotificationStatusSent, true},
		{enum.ResendEventEmailDeliveryDelayed, enum.NotificationStatusSent, true},
		{enum.ResendEventEmailDelivered, enum.NotificationStatusDelivered, true},
		{enum.ResendEventEmailFailed, enum.NotificationStatusFailed, true},
		{enum.ResendEventEmailBounced, enum.NotificationStatusFailed, true},
		{enum.ResendEventEmailComplained, enum.NotificationStatusFailed, true},
		{enum.ResendEventEmailSuppressed, enum.NotificationStatusFailed, true},
		{enum.ResendEventEmailOpened, "", false},
		{enum.ResendEventEmailClicked, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := MapResendEventStatus(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("MapResendEventStatus(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestMapWAHAAckStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		eventType  string
		ackName    string
		ack        int
		ackPresent bool
		want       enum.NotificationStatus
		wantOK     bool
	}{
		{"error", enum.WAHAEventMessageAck, enum.WAHAAckError, 0, false, enum.NotificationStatusFailed, true},
		{"pending", enum.WAHAEventMessageAck, enum.WAHAAckPending, 0, false, enum.NotificationStatusQueued, true},
		{"server", enum.WAHAEventMessageAck, enum.WAHAAckServer, 0, false, enum.NotificationStatusSent, true},
		{"device", enum.WAHAEventMessageAck, enum.WAHAAckDevice, 0, false, enum.NotificationStatusDelivered, true},
		{"read", enum.WAHAEventMessageAck, enum.WAHAAckRead, 0, false, enum.NotificationStatusDelivered, true},
		{"played", enum.WAHAEventMessageAck, enum.WAHAAckPlayed, 0, false, enum.NotificationStatusDelivered, true},
		{"ack-fallback", enum.WAHAEventMessageAck, "", 2, true, enum.NotificationStatusDelivered, true},
		{"missing-ack", enum.WAHAEventMessageAck, "", 0, false, "", false},
		{"other-event", "message", enum.WAHAAckRead, 3, true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MapWAHAAckStatus(tt.eventType, tt.ackName, tt.ack, tt.ackPresent)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("MapWAHAAckStatus() = %v, %v; want %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestMapWhatsocktEventStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType string
		want      enum.NotificationStatus
		wantOK    bool
	}{
		{enum.WhatsocktEventDispatching, enum.NotificationStatusProcessing, true},
		{enum.WhatsocktEventAccepted, enum.NotificationStatusSent, true},
		{enum.WhatsocktEventDelivered, enum.NotificationStatusDelivered, true},
		{enum.WhatsocktEventRead, enum.NotificationStatusDelivered, true},
		{enum.WhatsocktEventFailed, enum.NotificationStatusFailed, true},
		{enum.WhatsocktEventUnknown, enum.NotificationStatusFailed, true},
		{"message.inbound.received", "", false},
	}
	for _, tt := range tests {
		got, ok := MapWhatsocktEventStatus(tt.eventType)
		if got != tt.want || ok != tt.wantOK {
			t.Fatalf("MapWhatsocktEventStatus(%q) = %v, %v; want %v, %v", tt.eventType, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestApplyWhatsocktStatusFailure(t *testing.T) {
	t.Parallel()

	notification := &entity.Notification{Status: enum.NotificationStatusSent}
	applyWhatsocktStatus(notification, enum.NotificationStatusFailed, command.HandleWhatsocktWebhook{
		EventType: enum.WhatsocktEventFailed, ErrorCode: "send_failed", ErrorMessage: "socket closed",
	})
	if notification.Status != enum.NotificationStatusFailed || notification.LastError != "send_failed: socket closed" || notification.FailedAt == nil {
		t.Fatalf("unexpected notification: %+v", notification)
	}
}

func TestApplyWAHAStatusDoesNotRegressDeliveredToSent(t *testing.T) {
	t.Parallel()

	notification := &entity.Notification{Status: enum.NotificationStatusDelivered}
	applyWAHAStatus(notification, enum.NotificationStatusSent, command.HandleWAHAWebhook{EventType: enum.WAHAEventMessageAck, AckName: enum.WAHAAckServer})
	if notification.Status != enum.NotificationStatusDelivered {
		t.Fatalf("status = %v, want delivered", notification.Status)
	}
}

func TestApplyWAHAStatusFailureCapturesAck(t *testing.T) {
	t.Parallel()

	notification := &entity.Notification{Status: enum.NotificationStatusSent}
	applyWAHAStatus(notification, enum.NotificationStatusFailed, command.HandleWAHAWebhook{
		EventType: enum.WAHAEventMessageAck,
		AckName:   enum.WAHAAckError,
		Ack:       -1,
	})
	if notification.Status != enum.NotificationStatusFailed {
		t.Fatalf("status = %v, want failed", notification.Status)
	}
	if notification.LastError != "waha message ack ERROR" {
		t.Fatalf("last error = %q", notification.LastError)
	}
	if notification.FailedAt == nil {
		t.Fatal("failed_at is nil")
	}
}

func TestApplyResendStatusDoesNotRegressDeliveredToSent(t *testing.T) {
	t.Parallel()

	notification := &entity.Notification{Status: enum.NotificationStatusDelivered}
	applyResendStatus(notification, enum.NotificationStatusSent, command.HandleResendWebhook{EventType: enum.ResendEventEmailSent})
	if notification.Status != enum.NotificationStatusDelivered {
		t.Fatalf("status = %v, want delivered", notification.Status)
	}
}

func TestApplyResendStatusFailureCapturesBounceMessage(t *testing.T) {
	t.Parallel()

	notification := &entity.Notification{Status: enum.NotificationStatusSent}
	applyResendStatus(notification, enum.NotificationStatusFailed, command.HandleResendWebhook{
		EventType: enum.ResendEventEmailBounced,
		Data: types.JSONMap{
			"bounce": map[string]any{"message": "hard bounce"},
		},
	})
	if notification.Status != enum.NotificationStatusFailed {
		t.Fatalf("status = %v, want failed", notification.Status)
	}
	if notification.LastError != "hard bounce" {
		t.Fatalf("last error = %q", notification.LastError)
	}
	if notification.FailedAt == nil {
		t.Fatal("failed_at is nil")
	}
}
