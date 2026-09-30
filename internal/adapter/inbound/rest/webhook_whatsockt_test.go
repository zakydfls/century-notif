package rest

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"notif-gateway/internal/core/enum"
)

func TestVerifyWhatsocktWebhook(t *testing.T) {
	t.Parallel()

	body := []byte(`{"id":"event-1","type":"message.outbound.delivered","occurred_at":"2026-09-30T01:02:03Z","data":{"message_id":"message-1","status":"delivered"}}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	headers := http.Header{
		"X-Wa-Timestamp":   []string{timestamp},
		"X-Wa-Signature":   []string{whatsocktWebhookHMAC(timestamp, body, "secret")},
		"X-Wa-Delivery-Id": []string{"delivery-1"},
	}
	handler := &webhookREST{whatsocktWebhookCfg: WhatsocktWebhook{Secret: "secret", ValidateSignature: true}}
	if err := handler.verifyWhatsocktWebhook(body, headers); err != nil {
		t.Fatal(err)
	}
	headers.Set("X-WA-Signature", "sha256=invalid")
	if err := handler.verifyWhatsocktWebhook(body, headers); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestWhatsocktWebhookCommand(t *testing.T) {
	t.Parallel()

	body := []byte(`{"id":"event-1","type":"message.outbound.failed","occurred_at":"2026-09-30T01:02:03Z","data":{"message_id":"message-1","status":"failed","error_code":"send_failed","error_message":"socket closed"}}`)
	headers := http.Header{"X-Wa-Delivery-Id": []string{"delivery-1"}}
	cmd, err := whatsocktWebhookCommand(body, headers)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.EventID != "delivery-1" || cmd.EventType != enum.WhatsocktEventFailed || cmd.ProviderMessageID != "message-1" || cmd.ErrorCode != "send_failed" || cmd.ErrorMessage != "socket closed" {
		t.Fatalf("unexpected command: %+v", cmd)
	}
	if got := cmd.EventCreatedAt.Format(time.RFC3339); got != "2026-09-30T01:02:03Z" {
		t.Fatalf("occurred_at = %q", got)
	}
}
