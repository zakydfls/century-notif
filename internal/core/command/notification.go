package command

// ListNotifications filters for GET /notifications.
type ListNotifications struct {
	UserID      string `query:"user_id" validate:"omitempty,uuid"`
	ServiceName string `query:"service_name" validate:"omitempty,max=128"`
	Channel     string `query:"channel" validate:"omitempty,oneof=sms whatsapp email"`
	Provider    string `query:"provider" validate:"omitempty,oneof=twilio alibaba waha whatsockt godaddy resend"`
	Status      string `query:"status" validate:"omitempty,oneof=queued processing sent delivered retrying failed pending"`
	Recipient   string `query:"recipient" validate:"omitempty"`
}
