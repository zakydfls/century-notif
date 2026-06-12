package event

import (
	"testing"

	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
)

func TestSendNotificationPayload_HasReferenceOnly(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	p := SendNotificationPayload{
		NotificationID: id,
		Channel:        enum.ChannelSMS,
	}
	if !p.HasReferenceOnly() {
		t.Fatal("expected reference-only payload")
	}
	if !p.HasNotificationID() {
		t.Fatal("expected notification id")
	}
	if p.HasCreateFields() {
		t.Fatal("reference-only should not have create fields")
	}
}

func TestSendNotificationPayload_HasCreateFields(t *testing.T) {
	t.Parallel()

	p := SendNotificationPayload{
		Channel:   enum.ChannelEmail,
		Recipient: "a@b.com",
		Body:      "hi",
		Metadata:  map[string]any{enum.EmailMetaSubject: "Sub"},
	}
	if !p.HasCreateFields() {
		t.Fatal("expected create fields")
	}
	if p.HasReferenceOnly() {
		t.Fatal("create payload is not reference-only")
	}
}
