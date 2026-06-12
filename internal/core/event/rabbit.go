package event

import (
	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
)

type Type string

const (
	TypeNotificationSend Type = "notification.send"
)

func (t Type) String() string {
	return string(t)
}

const (
	ExchangeSMS      = "notif.sms.exchange"
	ExchangeWhatsApp = "notif.whatsapp.exchange"
	ExchangeEmail    = "notif.email.exchange"
	ExchangeDLX      = "notif.dlx.exchange"

	QueueSMSSend       = "notif.sms.send"
	QueueSMSRetry      = "notif.sms.retry"
	QueueSMSDeadLetter = "notif.sms.dlq"
	QueueWhatsAppSend  = "notif.whatsapp.send"
	QueueWhatsAppRetry = "notif.whatsapp.retry"
	QueueWhatsAppDLQ   = "notif.whatsapp.dlq"
	QueueEmailSend     = "notif.email.send"
	QueueEmailRetry    = "notif.email.retry"
	QueueEmailDLQ      = "notif.email.dlq"

	RoutingKeySend       = "send"
	RoutingKeyRetry      = "retry"
	RoutingKeyDeadLetter = "dead_letter"
)

type OutboxMessage struct {
	ID      uuid.UUID               `json:"id"`
	Type    Type                    `json:"type"`
	Channel enum.Channel            `json:"channel"`
	Payload SendNotificationPayload `json:"payload"`
}

func ExchangeForChannel(channel enum.Channel) string {
	switch channel {
	case enum.ChannelWhatsApp:
		return ExchangeWhatsApp
	case enum.ChannelEmail:
		return ExchangeEmail
	default:
		return ExchangeSMS
	}
}

func QueueForChannel(channel enum.Channel) string {
	switch channel {
	case enum.ChannelWhatsApp:
		return QueueWhatsAppSend
	case enum.ChannelEmail:
		return QueueEmailSend
	default:
		return QueueSMSSend
	}
}

func RetryQueueForChannel(channel enum.Channel) string {
	switch channel {
	case enum.ChannelWhatsApp:
		return QueueWhatsAppRetry
	case enum.ChannelEmail:
		return QueueEmailRetry
	default:
		return QueueSMSRetry
	}
}

func DeadLetterQueueForChannel(channel enum.Channel) string {
	switch channel {
	case enum.ChannelWhatsApp:
		return QueueWhatsAppDLQ
	case enum.ChannelEmail:
		return QueueEmailDLQ
	default:
		return QueueSMSDeadLetter
	}
}
