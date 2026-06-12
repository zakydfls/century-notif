package enum

type NotificationStatus string

const (
	NotificationStatusQueued     NotificationStatus = "queued"
	NotificationStatusProcessing NotificationStatus = "processing"
	NotificationStatusSent       NotificationStatus = "sent"
	NotificationStatusDelivered  NotificationStatus = "delivered"
	NotificationStatusFailed     NotificationStatus = "failed"
	NotificationStatusRetrying   NotificationStatus = "retrying"
)

func (s NotificationStatus) String() string {
	return string(s)
}

func (s NotificationStatus) IsTerminal() bool {
	return s == NotificationStatusDelivered || s == NotificationStatusFailed
}

type AlibabaMode string

const (
	AlibabaModeGlobe AlibabaMode = "globe"
	AlibabaModeChina AlibabaMode = "china"
)

func (m AlibabaMode) String() string {
	return string(m)
}
