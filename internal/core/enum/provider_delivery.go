package enum

// SupportsDeliveryWebhook reports whether delivery confirmation arrives via HTTP webhook after send.
func (p Provider) SupportsDeliveryWebhook(channel Channel) bool {
	switch channel {
	case ChannelSMS:
		return p == ProviderTwilio
	case ChannelWhatsApp:
		return p == ProviderTwilio || p == ProviderWAHA
	case ChannelEmail:
		return p == ProviderResend
	default:
		return false
	}
}

// StatusAfterSuccessfulSend returns the notification status immediately after the provider accepts the send.
func (p Provider) StatusAfterSuccessfulSend(channel Channel) NotificationStatus {
	if p.SupportsDeliveryWebhook(channel) {
		return NotificationStatusSent
	}
	return NotificationStatusDelivered
}
