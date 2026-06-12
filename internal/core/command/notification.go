package command

// ListNotifications filters for GET /notifications.
type ListNotifications struct {
	UserID    string `query:"user_id" validate:"omitempty,uuid"`
	Channel   string `query:"channel" validate:"omitempty,oneof=sms whatsapp email"`
	Provider  string `query:"provider" validate:"omitempty,oneof=twilio alibaba waha godaddy resend"`
	Status    string `query:"status" validate:"omitempty,oneof=queued processing sent delivered retrying failed pending"`
	Recipient string `query:"recipient" validate:"omitempty"`
}
