package rest

import (
	"log/slog"

	"notif-gateway/internal/core/service"
)

// TwilioWebhook holds settings for validating Twilio status POSTs per
// https://www.twilio.com/docs/usage/webhooks/webhooks-security
type TwilioWebhook struct {
	AuthToken         string
	BaseURL           string // trimmed origin from TWILIO_WEBHOOK_BASE_URL — must match CreateMessage StatusCallback prefix
	ValidateSignature bool
}

type ResendWebhook struct {
	Secret            string
	ValidateSignature bool
}

type WAHAWebhook struct {
	HMACKey           string
	ValidateSignature bool
}

// Handler composes REST modules and exposes the root HTTP handler.
type Handler struct {
	notifications      NotificationREST
	webhooks           WebhookREST
	log                *slog.Logger
	notificationAPIKey string
}

func NewHandler(
	notifications service.Notification,
	webhooks service.Webhook,
	log *slog.Logger,
	notificationAPIKey string,
	twilio TwilioWebhook,
	waha WAHAWebhook,
	resend ResendWebhook,
) *Handler {
	return &Handler{
		notifications:      NewNotificationREST(notifications),
		webhooks:           NewWebhookREST(webhooks, twilio, waha, resend),
		log:                log,
		notificationAPIKey: notificationAPIKey,
	}
}
