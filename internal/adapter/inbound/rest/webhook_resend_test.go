package rest

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	svix "github.com/svix/svix-webhooks/go"
)

func TestResendWebhookCommand(t *testing.T) {
	t.Parallel()

	body := []byte(`{"type":"email.delivered","created_at":"2026-06-12T10:11:12.123Z","data":{"email_id":"email_123","subject":"Hi"}}`)
	header := http.Header{}
	header.Set("svix-id", "msg_123")
	header.Set("svix-timestamp", "1700000000")
	header.Set("svix-signature", "v1,signature")

	cmd, err := resendWebhookCommand(body, header)
	if err != nil {
		t.Fatalf("resendWebhookCommand: %v", err)
	}
	if cmd.EventID != "msg_123" {
		t.Fatalf("event id = %q", cmd.EventID)
	}
	if cmd.EventType != "email.delivered" {
		t.Fatalf("event type = %q", cmd.EventType)
	}
	if cmd.ProviderMessageID != "email_123" {
		t.Fatalf("provider message id = %q", cmd.ProviderMessageID)
	}
	if cmd.Data["subject"] != "Hi" {
		t.Fatalf("data = %v", cmd.Data)
	}
	if cmd.Headers["svix_id"] != "msg_123" {
		t.Fatalf("headers = %v", cmd.Headers)
	}
	if cmd.EventCreatedAt.IsZero() {
		t.Fatal("event created at is zero")
	}
}

func TestResendWebhookCommandFallbackEventID(t *testing.T) {
	t.Parallel()

	body := []byte(`{"type":"email.sent","data":{"email_id":"email_123"}}`)
	cmd, err := resendWebhookCommand(body, http.Header{})
	if err != nil {
		t.Fatalf("resendWebhookCommand: %v", err)
	}
	if cmd.EventID == "" {
		t.Fatal("event id is empty")
	}
	if cmd.EventID[:7] != "sha256:" {
		t.Fatalf("event id = %q, want sha256 fallback", cmd.EventID)
	}
}

func TestVerifyResendWebhook(t *testing.T) {
	t.Parallel()

	secret := "whsec_dGVzdF9zZWNyZXQ="
	body := []byte(`{"type":"email.sent","data":{"email_id":"email_123"}}`)
	timestamp := time.Now()
	wh, err := svix.NewWebhook(secret)
	if err != nil {
		t.Fatalf("NewWebhook: %v", err)
	}
	signature, err := wh.Sign("msg_123", timestamp, body)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	header := http.Header{}
	header.Set("svix-id", "msg_123")
	header.Set("svix-timestamp", strconv.FormatInt(timestamp.Unix(), 10))
	header.Set("svix-signature", signature)

	handler := &webhookREST{resendWebhookCfg: ResendWebhook{Secret: secret, ValidateSignature: true}}
	if err := handler.verifyResendWebhook(body, header); err != nil {
		t.Fatalf("verifyResendWebhook: %v", err)
	}

	header.Set("svix-signature", "v1,invalid")
	if err := handler.verifyResendWebhook(body, header); err == nil {
		t.Fatal("verifyResendWebhook nil error, want invalid signature error")
	}
}
