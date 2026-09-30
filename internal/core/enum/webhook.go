package enum

// TwilioMessageStatus values for MessageStatus / SmsStatus on outbound Twilio webhooks.
// See https://www.twilio.com/docs/messaging/guides/outbound-message-status-in-status-callbacks
const (
	TwilioMessageStatusQueued      = "queued"
	TwilioMessageStatusSending     = "sending"
	TwilioMessageStatusSent        = "sent"
	TwilioMessageStatusDelivered   = "delivered"
	TwilioMessageStatusUndelivered = "undelivered"
	TwilioMessageStatusFailed      = "failed"
	TwilioMessageStatusRead        = "read"
	TwilioMessageStatusAccepted    = "accepted"
	TwilioMessageStatusScheduled   = "scheduled"
	TwilioMessageStatusReceiving   = "receiving"
	TwilioMessageStatusCanceled    = "canceled"
	TwilioMessageStatusCancelled   = "cancelled"
)

// Whatsockt outbound webhook event types.
const (
	WhatsocktEventDispatching = "message.outbound.dispatching"
	WhatsocktEventAccepted    = "message.outbound.accepted"
	WhatsocktEventDelivered   = "message.outbound.delivered"
	WhatsocktEventRead        = "message.outbound.read"
	WhatsocktEventFailed      = "message.outbound.failed"
	WhatsocktEventUnknown     = "message.outbound.unknown"
)

// JSON webhook status strings used by non-Twilio providers (e.g. Alibaba/WAHA).
// not fully documented yet.
const (
	WebhookStatusSuccess               = "success"
	WebhookStatusSuccessfullyDelivered = "successfully_delivered"
	WebhookStatusError                 = "error"
	WebhookStatusRejected              = "rejected"
)

// WAHA webhook event and ack values.
// See https://waha.devlike.pro/docs/how-to/events/#messageack
const (
	WAHAEventMessageAck = "message.ack"

	WAHAAckError   = "ERROR"
	WAHAAckPending = "PENDING"
	WAHAAckServer  = "SERVER"
	WAHAAckDevice  = "DEVICE"
	WAHAAckRead    = "READ"
	WAHAAckPlayed  = "PLAYED"
)

// Resend email webhook event types.
// See https://resend.com/docs/webhooks/event-types
const (
	ResendEventEmailSent            = "email.sent"
	ResendEventEmailDelivered       = "email.delivered"
	ResendEventEmailDeliveryDelayed = "email.delivery_delayed"
	ResendEventEmailFailed          = "email.failed"
	ResendEventEmailBounced         = "email.bounced"
	ResendEventEmailComplained      = "email.complained"
	ResendEventEmailSuppressed      = "email.suppressed"
	ResendEventEmailOpened          = "email.opened"
	ResendEventEmailClicked         = "email.clicked"
	ResendEventEmailScheduled       = "email.scheduled"
	ResendEventEmailReceived        = "email.received"
)
