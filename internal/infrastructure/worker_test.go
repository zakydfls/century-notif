package infrastructure

import (
	"testing"

	"notif-gateway/internal/core/enum"
)

func TestChannelWorkers_ForChannel(t *testing.T) {
	t.Parallel()

	workers := NewChannelWorkers(ChannelWorkerConfig{Prefetch: 16, Concurrency: 8, MaxRetry: 3})

	email := workers.ForChannel(enum.ChannelEmail)
	if email.Concurrency != 5 || email.Prefetch != 5 || email.MaxRetry != 3 {
		t.Fatalf("email = %+v", email)
	}

	sms := workers.ForChannel(enum.ChannelSMS)
	if sms.Concurrency != 10 || sms.Prefetch != 10 || sms.MaxRetry != 3 {
		t.Fatalf("sms = %+v", sms)
	}

	whatsapp := workers.ForChannel(enum.ChannelWhatsApp)
	if whatsapp.Concurrency != 6 || whatsapp.Prefetch != 6 || whatsapp.MaxRetry != 3 {
		t.Fatalf("whatsapp = %+v", whatsapp)
	}
}
