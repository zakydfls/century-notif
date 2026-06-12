package rest

import (
	"net/http"
	"testing"
	"time"
)

func TestWAHAWebhookHMACMatchesDocsExample(t *testing.T) {
	t.Parallel()

	body := []byte(`{"event":"message","session":"default","engine":"WEBJS"}`)
	got := wahaWebhookHMAC(body, "my-secret-key")
	want := "208f8a55dde9e05519e898b10b89bf0d0b3b0fdf11fdbf09b6b90476301b98d8097c462b2b17a6ce93b6b47a136cf2e78a33a63f6752c2c1631777076153fa89"
	if got != want {
		t.Fatalf("hmac = %q, want %q", got, want)
	}
}

func TestVerifyWAHAWebhook(t *testing.T) {
	t.Parallel()

	body := []byte(`{"event":"message.ack","session":"default","payload":{"id":"true_11111111111@c.us_4CC","ack":3,"ackName":"READ"}}`)
	key := "secret"
	header := http.Header{}
	header.Set("X-Webhook-Hmac-Algorithm", "sha512")
	header.Set("X-Webhook-Hmac", wahaWebhookHMAC(body, key))

	handler := &webhookREST{wahaWebhookCfg: WAHAWebhook{HMACKey: key, ValidateSignature: true}}
	if err := handler.verifyWAHAWebhook(body, header); err != nil {
		t.Fatalf("verifyWAHAWebhook: %v", err)
	}

	header.Set("X-Webhook-Hmac", "invalid")
	if err := handler.verifyWAHAWebhook(body, header); err == nil {
		t.Fatal("verifyWAHAWebhook nil error, want invalid hmac error")
	}
}

func TestWAHAWebhookCommand(t *testing.T) {
	t.Parallel()

	body := []byte(`{"id":"evt_111","timestamp":1741249702485,"event":"message.ack","session":"default","engine":"WEBJS","payload":{"id":"true_11111111111@c.us_4CC","from":"11111111111@c.us","fromMe":true,"ack":3,"ackName":"READ"}}`)
	header := http.Header{}
	header.Set("X-Webhook-Request-Id", "req_123")
	header.Set("X-Webhook-Timestamp", "1741249702485")
	header.Set("X-Webhook-Hmac-Algorithm", "sha512")
	header.Set("X-Webhook-Hmac", "signature")

	cmd, err := wahaWebhookCommand(body, header)
	if err != nil {
		t.Fatalf("wahaWebhookCommand: %v", err)
	}
	if cmd.EventID != "evt_111" {
		t.Fatalf("event id = %q", cmd.EventID)
	}
	if cmd.EventType != "message.ack" {
		t.Fatalf("event type = %q", cmd.EventType)
	}
	if cmd.Session != "default" {
		t.Fatalf("session = %q", cmd.Session)
	}
	if cmd.ProviderMessageID != "true_11111111111@c.us_4CC" {
		t.Fatalf("provider message id = %q", cmd.ProviderMessageID)
	}
	if cmd.AckName != "READ" || cmd.Ack != 3 || !cmd.AckPresent {
		t.Fatalf("ack = %q/%d/%v", cmd.AckName, cmd.Ack, cmd.AckPresent)
	}
	want := time.UnixMilli(1741249702485)
	if !cmd.EventCreatedAt.Equal(want) {
		t.Fatalf("created at = %v, want %v", cmd.EventCreatedAt, want)
	}
	if cmd.Headers["x_webhook_request_id"] != "req_123" {
		t.Fatalf("headers = %v", cmd.Headers)
	}
}
