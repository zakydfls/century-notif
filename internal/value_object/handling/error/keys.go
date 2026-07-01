package apperror

const (
	KeyInternal       = "errors.internal"
	KeyUnauthorized   = "errors.auth.unauthorized"
	KeyForbidden      = "errors.auth.forbidden"
	KeyRecordNotFound = "errors.record_not_found"
	KeyInvalidUUID    = "errors.id.invalid_uuid"

	KeyRequestBodyRequired = "errors.request.body_required"
	KeyInvalidJSONBody     = "errors.request.invalid_json"
	KeyRequestRequired     = "errors.request.required"
	KeyInvalidQueryParam   = "errors.request.invalid_query"
	KeyValidationFailed    = "errors.validation.failed"

	KeyEnqueueSingleRecipient    = "notification.enqueue.single_recipient_required"
	KeyServiceNameInvalid        = "notification.service_name.invalid"
	KeyInvalidChannel            = "notification.channel.invalid"
	KeyMissingContent            = "notification.content.required"
	KeyEmailSubjectRequired      = "notification.email.subject_required"
	KeyEmailContentTypeInvalid   = "notification.email.content_type_invalid"
	KeyPayloadSingleNotification = "notification.payload.single_notification_required"

	KeyWebhookTargetMissing  = "webhook.target.missing"
	KeyWebhookBodyReadFailed = "webhook.request.body_read_failed"
	KeyWebhookInvalidForm    = "webhook.request.invalid_form"
	KeyWebhookInvalidJSON    = "webhook.request.invalid_json"
	KeyWebhookInvalidSign    = "webhook.request.invalid_signature"

	KeyProviderChannelNotSupported = "provider.channel.not_supported"
	KeyProviderNotFound            = "provider.not_found"
	KeyProviderSendFailed          = "provider.send_failed"

	KeyGodaddySMTPHostRequired = "provider.godaddy.smtp_host_required"
	KeyResendAPIKeyRequired    = "provider.resend.api_key_required"
	KeyResendWebhookRequired   = "provider.resend.webhook_secret_required"
	KeyEmailFromRequired       = "provider.email.from_required"
	KeyEmailInvalidFrom        = "provider.email.invalid_from"
	KeyEmailInvalidRecipient   = "provider.email.invalid_recipient"
	KeyEmailInvalidReplyTo     = "provider.email.invalid_reply_to"
	KeyEmailInvalidCC          = "provider.email.invalid_cc"
	KeyEmailInvalidBCC         = "provider.email.invalid_bcc"

	KeyTwilioSMSSenderRequired      = "provider.twilio.sms_sender_required"
	KeyTwilioWhatsAppSenderRequired = "provider.twilio.whatsapp_sender_required"

	KeyAlibabaTemplateRequired    = "provider.alibaba.template_required"
	KeyAlibabaClientNotConfigured = "provider.alibaba.client_not_configured"

	KeyWAHASendFailed             = "provider.waha.send_failed"
	KeyWAHAWebhookHMACKeyRequired = "provider.waha.webhook_hmac_key_required"
	KeyNotImplemented             = "errors.not_implemented"
)
