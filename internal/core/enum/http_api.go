package enum

// HTTPAPIPrefix is the mount path for versioned REST API routes (e.g. /api/v1/notifications).
const HTTPAPIPrefix = "/api/v1"

// TwilioWebhookPath* are appended to TWILIO_WEBHOOK_BASE_URL (origin only) when setting CreateMessage StatusCallback.
const (
	TwilioWebhookPathSMS      = HTTPAPIPrefix + "/webhooks/twilio/sms"
	TwilioWebhookPathWhatsApp = HTTPAPIPrefix + "/webhooks/twilio/whatsapp"
	WAHAWebhookPathWhatsApp   = HTTPAPIPrefix + "/webhooks/waha/whatsapp"
	WhatsocktWebhookPath      = HTTPAPIPrefix + "/webhooks/whatsockt/whatsapp"
	ResendWebhookPath         = HTTPAPIPrefix + "/webhooks/resend"
)
