package enum

import "testing"

func TestProvider_SupportsDeliveryWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		channel  Channel
		provider Provider
		want     bool
	}{
		{ChannelSMS, ProviderTwilio, true},
		{ChannelSMS, ProviderAlibaba, false},
		{ChannelWhatsApp, ProviderTwilio, true},
		{ChannelWhatsApp, ProviderWAHA, true},
		{ChannelEmail, ProviderGoDaddy, false},
		{ChannelEmail, ProviderResend, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.channel)+"_"+string(tt.provider), func(t *testing.T) {
			if got := tt.provider.SupportsDeliveryWebhook(tt.channel); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_StatusAfterSuccessfulSend(t *testing.T) {
	t.Parallel()

	if got := ProviderTwilio.StatusAfterSuccessfulSend(ChannelSMS); got != NotificationStatusSent {
		t.Fatalf("twilio sms got %v", got)
	}
	if got := ProviderWAHA.StatusAfterSuccessfulSend(ChannelWhatsApp); got != NotificationStatusSent {
		t.Fatalf("waha whatsapp got %v", got)
	}
	if got := ProviderGoDaddy.StatusAfterSuccessfulSend(ChannelEmail); got != NotificationStatusDelivered {
		t.Fatalf("godaddy email got %v", got)
	}
	if got := ProviderResend.StatusAfterSuccessfulSend(ChannelEmail); got != NotificationStatusSent {
		t.Fatalf("resend email got %v", got)
	}
}
