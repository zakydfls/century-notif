package whatsockt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/infrastructure"
)

func TestProviderSend(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != sendMessagePath || r.Header.Get("X-API-Key") != "secret" || r.Header.Get("Idempotency-Key") != "notification-1" {
			t.Fatalf("unexpected request: path=%q api_key=%q idempotency=%q", r.URL.Path, r.Header.Get("X-API-Key"), r.Header.Get("Idempotency-Key"))
		}
		var body sendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.DeviceID != "device-1" || body.Recipient != "+628111111111" || body.Content.Type != "text" || body.Content.Text != "hello" {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"message-1","status":"queued"}`))
	}))
	defer server.Close()

	provider := NewProvider(infrastructure.WhatsocktConfig{
		BaseURL: server.URL, APIKey: "secret", DeviceID: "device-1", Timeout: time.Second,
	})
	result, err := provider.Send(context.Background(), command.ProviderSend{
		NotificationID: "notification-1", Recipient: "whatsapp:+628111111111@c.us", Body: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "message-1" || result.Raw["status"] != "queued" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestProviderSendError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_message","message":"recipient is invalid"}}`))
	}))
	defer server.Close()

	provider := NewProvider(infrastructure.WhatsocktConfig{
		BaseURL: server.URL, APIKey: "secret", DeviceID: "device-1", Timeout: time.Second,
	})
	_, err := provider.Send(context.Background(), command.ProviderSend{Recipient: "+62", Body: "hello"})
	if err == nil || err.Error() != "whatsockt send failed (http 400): invalid_message: recipient is invalid" {
		t.Fatalf("unexpected error: %v", err)
	}
}
