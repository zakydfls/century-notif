package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	apperror "notif-gateway/internal/value_object/handling/error"
)

// Webhook is the application port for provider status callbacks.
type Webhook interface {
	Handle(ctx context.Context, cmd command.HandleWebhook) error
	HandleWAHA(ctx context.Context, cmd command.HandleWAHAWebhook) error
	HandleResend(ctx context.Context, cmd command.HandleResendWebhook) error
}

var _ Webhook = (*webhook)(nil)

type webhook struct {
	notif  repository.Notification
	events repository.WebhookEvent
	txm    *repository.TxManager
}

func NewWebhookHandler(notif repository.Notification, events repository.WebhookEvent, txm *repository.TxManager) Webhook {
	return &webhook{notif: notif, events: events, txm: txm}
}

func (h *webhook) Handle(ctx context.Context, cmd command.HandleWebhook) error {
	notification, err := h.findNotification(ctx, cmd)
	if err != nil {
		return err
	}
	now := time.Now()
	status := MapProviderStatus(cmd.ExternalStatus)
	notification.Status = status

	switch status {
	case enum.NotificationStatusFailed:
		notification.LastError = formatProviderWebhookFailure(cmd.Provider, cmd)
		notification.FailedAt = &now
	case enum.NotificationStatusDelivered:
		notification.LastError = ""
		notification.DeliveredAt = &now
	default:
	}

	return h.notif.Save(ctx, notification)
}

func (h *webhook) HandleResend(ctx context.Context, cmd command.HandleResendWebhook) error {
	if h.events == nil || h.txm == nil {
		return apperror.ErrInternal(errors.New("resend webhook handler is not configured"))
	}
	return h.txm.Do(ctx, func(tx *gorm.DB) error {
		return h.handleResendWithRepos(ctx, cmd, h.notif.WithTrx(tx), h.events.WithTrx(tx))
	})
}

func (h *webhook) HandleWAHA(ctx context.Context, cmd command.HandleWAHAWebhook) error {
	if h.events == nil || h.txm == nil {
		return apperror.ErrInternal(errors.New("waha webhook handler is not configured"))
	}
	return h.txm.Do(ctx, func(tx *gorm.DB) error {
		return h.handleWAHAWithRepos(ctx, cmd, h.notif.WithTrx(tx), h.events.WithTrx(tx))
	})
}

func (h *webhook) handleWAHAWithRepos(ctx context.Context, cmd command.HandleWAHAWebhook, notif repository.Notification, events repository.WebhookEvent) error {
	now := time.Now()
	eventCreatedAt := cmd.EventCreatedAt
	event := &entity.NotificationWebhookEvent{
		Provider:          enum.ProviderWAHA,
		EventID:           strings.TrimSpace(cmd.EventID),
		EventType:         strings.TrimSpace(cmd.EventType),
		ProviderMessageID: strings.TrimSpace(cmd.ProviderMessageID),
		ReceivedAt:        now,
		Payload:           commandJSONMap(cmd.Payload),
		Headers:           commandJSONMap(cmd.Headers),
	}
	if !eventCreatedAt.IsZero() {
		event.EventCreatedAt = &eventCreatedAt
	}
	if err := events.Create(ctx, event); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil
		}
		return err
	}

	status, ok := MapWAHAAckStatus(cmd.EventType, cmd.AckName, cmd.Ack, cmd.AckPresent)
	if !ok {
		event.ProcessedAt = &now
		return events.Save(ctx, event)
	}
	event.Status = status

	if event.ProviderMessageID == "" {
		event.ErrorMessage = "waha webhook message id is missing"
		event.ProcessedAt = &now
		return events.Save(ctx, event)
	}

	notification := &entity.Notification{
		Provider:          enum.ProviderWAHA,
		ProviderMessageID: event.ProviderMessageID,
	}
	if err := notif.Find(ctx, notification); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			event.ErrorMessage = "notification not found for waha message id: " + event.ProviderMessageID
			event.ProcessedAt = &now
			return events.Save(ctx, event)
		}
		return err
	}
	event.NotificationID = &notification.ID
	applyWAHAStatus(notification, status, cmd)
	event.ProcessedAt = &now

	if err := notif.Save(ctx, notification); err != nil {
		return err
	}
	return events.Save(ctx, event)
}

func (h *webhook) handleResendWithRepos(ctx context.Context, cmd command.HandleResendWebhook, notif repository.Notification, events repository.WebhookEvent) error {
	now := time.Now()
	eventCreatedAt := cmd.EventCreatedAt
	event := &entity.NotificationWebhookEvent{
		Provider:          enum.ProviderResend,
		EventID:           strings.TrimSpace(cmd.EventID),
		EventType:         strings.TrimSpace(cmd.EventType),
		ProviderMessageID: strings.TrimSpace(cmd.ProviderMessageID),
		ReceivedAt:        now,
		Payload:           commandJSONMap(cmd.Payload),
		Headers:           commandJSONMap(cmd.Headers),
	}
	if !eventCreatedAt.IsZero() {
		event.EventCreatedAt = &eventCreatedAt
	}
	if err := events.Create(ctx, event); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil
		}
		return err
	}

	status, ok := MapResendEventStatus(cmd.EventType)
	if !ok {
		event.ProcessedAt = &now
		return events.Save(ctx, event)
	}
	event.Status = status

	if event.ProviderMessageID == "" {
		event.ErrorMessage = "resend webhook email_id is missing"
		event.ProcessedAt = &now
		return events.Save(ctx, event)
	}

	notification := &entity.Notification{
		Provider:          enum.ProviderResend,
		ProviderMessageID: event.ProviderMessageID,
	}
	if err := notif.Find(ctx, notification); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			event.ErrorMessage = "notification not found for resend email_id: " + event.ProviderMessageID
			event.ProcessedAt = &now
			return events.Save(ctx, event)
		}
		return err
	}
	event.NotificationID = &notification.ID
	applyResendStatus(notification, status, cmd)
	event.ProcessedAt = &now

	if err := notif.Save(ctx, notification); err != nil {
		return err
	}
	return events.Save(ctx, event)
}

func formatProviderWebhookFailure(p enum.Provider, c command.HandleWebhook) string {
	switch p {
	case enum.ProviderTwilio:
		return formatTwilioWebhookError(c.ErrorCode, c.ErrorMessage, c.ChannelStatusMessage)
	default:
		if msg := strings.TrimSpace(c.ErrorMessage); msg != "" {
			return msg
		}
		return strings.TrimSpace(c.ErrorCode)
	}
}

func formatWAHAWebhookFailure(cmd command.HandleWAHAWebhook) string {
	for _, key := range []string{"message", "reason", "error"} {
		if msg := strings.TrimSpace(stringFromJSON(cmd.Data, key)); msg != "" {
			return msg
		}
	}
	ack := strings.TrimSpace(cmd.AckName)
	if ack == "" && cmd.Ack != 0 {
		ack = fmt.Sprintf("ack=%d", cmd.Ack)
	}
	if ack == "" {
		return strings.TrimSpace(cmd.EventType)
	}
	return "waha message ack " + ack
}

func formatResendWebhookFailure(cmd command.HandleResendWebhook) string {
	for _, key := range []string{"message", "reason", "error"} {
		if msg := strings.TrimSpace(stringFromJSON(cmd.Data, key)); msg != "" {
			return msg
		}
	}
	switch cmd.EventType {
	case enum.ResendEventEmailBounced:
		if msg := nestedStringFromJSON(cmd.Data, "bounce", "message"); msg != "" {
			return msg
		}
		return "email bounced"
	case enum.ResendEventEmailComplained:
		return "recipient marked email as spam"
	case enum.ResendEventEmailSuppressed:
		return "email suppressed by resend"
	default:
		return strings.TrimSpace(cmd.EventType)
	}
}

func formatTwilioWebhookError(code, errMsg, channelMsg string) string {
	code = strings.TrimSpace(code)
	errMsg = strings.TrimSpace(errMsg)
	channelMsg = strings.TrimSpace(channelMsg)

	var parts []string
	if code != "" {
		parts = append(parts, code)
	}
	if errMsg != "" {
		parts = append(parts, errMsg)
	}
	if channelMsg != "" && channelMsg != errMsg {
		parts = append(parts, channelMsg)
	}
	return strings.Join(parts, ": ")
}

func (h *webhook) findNotification(ctx context.Context, cmd command.HandleWebhook) (*entity.Notification, error) {
	notification := new(entity.Notification)
	switch {
	case cmd.NotificationID != uuid.Nil:
		notification.ID = cmd.NotificationID
	case cmd.ProviderMessageID != "":
		notification.Provider = cmd.Provider
		notification.ProviderMessageID = cmd.ProviderMessageID
	default:
		return nil, apperror.ErrBadRequest(nil, "webhook target is missing", apperror.KeyWebhookTargetMissing)
	}
	if err := h.notif.Find(ctx, notification); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.ErrNotFound(err, err.Error(), apperror.KeyRecordNotFound)
		}
		return nil, err
	}
	return notification, nil
}

// MapProviderStatus maps provider webhook payloads to gateway notification statuses. Twilio Programmable Messaging
// values are documented at https://www.twilio.com/docs/messaging/guides/outbound-message-status-in-status-callbacks — other aliases cover JSON webhooks (Alibaba/WAHA).
func MapProviderStatus(status string) enum.NotificationStatus {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case enum.TwilioMessageStatusDelivered, enum.TwilioMessageStatusRead,
		enum.WebhookStatusSuccess, enum.WebhookStatusSuccessfullyDelivered:
		return enum.NotificationStatusDelivered
	case enum.TwilioMessageStatusFailed, enum.TwilioMessageStatusUndelivered,
		enum.WebhookStatusError, enum.WebhookStatusRejected,
		enum.TwilioMessageStatusCanceled, enum.TwilioMessageStatusCancelled:
		return enum.NotificationStatusFailed
	case enum.TwilioMessageStatusQueued, enum.TwilioMessageStatusAccepted, enum.TwilioMessageStatusScheduled:
		return enum.NotificationStatusQueued
	case enum.TwilioMessageStatusSent:
		return enum.NotificationStatusSent
	case enum.TwilioMessageStatusSending, enum.TwilioMessageStatusReceiving:
		return enum.NotificationStatusProcessing
	default:
		return enum.NotificationStatusProcessing
	}
}

func MapResendEventStatus(eventType string) (enum.NotificationStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case enum.ResendEventEmailScheduled:
		return enum.NotificationStatusQueued, true
	case enum.ResendEventEmailSent, enum.ResendEventEmailDeliveryDelayed:
		return enum.NotificationStatusSent, true
	case enum.ResendEventEmailDelivered:
		return enum.NotificationStatusDelivered, true
	case enum.ResendEventEmailFailed, enum.ResendEventEmailBounced,
		enum.ResendEventEmailComplained, enum.ResendEventEmailSuppressed:
		return enum.NotificationStatusFailed, true
	default:
		return "", false
	}
}

func MapWAHAAckStatus(eventType, ackName string, ack int, ackPresent bool) (enum.NotificationStatus, bool) {
	if !strings.EqualFold(strings.TrimSpace(eventType), enum.WAHAEventMessageAck) {
		return "", false
	}
	switch strings.ToUpper(strings.TrimSpace(ackName)) {
	case enum.WAHAAckError:
		return enum.NotificationStatusFailed, true
	case enum.WAHAAckPending:
		return enum.NotificationStatusQueued, true
	case enum.WAHAAckServer:
		return enum.NotificationStatusSent, true
	case enum.WAHAAckDevice, enum.WAHAAckRead, enum.WAHAAckPlayed:
		return enum.NotificationStatusDelivered, true
	}
	if !ackPresent {
		return "", false
	}
	switch ack {
	case -1:
		return enum.NotificationStatusFailed, true
	case 0:
		return enum.NotificationStatusQueued, true
	case 1:
		return enum.NotificationStatusSent, true
	case 2, 3, 4:
		return enum.NotificationStatusDelivered, true
	default:
		return "", false
	}
}

func applyWAHAStatus(notification *entity.Notification, status enum.NotificationStatus, cmd command.HandleWAHAWebhook) {
	now := time.Now()
	if !shouldApplyProviderStatus(notification.Status, status) {
		return
	}
	notification.Status = status
	switch status {
	case enum.NotificationStatusQueued:
	case enum.NotificationStatusSent:
		if notification.SentAt == nil {
			notification.SentAt = &now
		}
		notification.LastError = ""
	case enum.NotificationStatusDelivered:
		notification.LastError = ""
		if notification.SentAt == nil {
			notification.SentAt = &now
		}
		notification.DeliveredAt = &now
	case enum.NotificationStatusFailed:
		notification.LastError = formatWAHAWebhookFailure(cmd)
		notification.FailedAt = &now
	default:
	}
}

func applyResendStatus(notification *entity.Notification, status enum.NotificationStatus, cmd command.HandleResendWebhook) {
	now := time.Now()
	if !shouldApplyProviderStatus(notification.Status, status) {
		return
	}
	notification.Status = status
	switch status {
	case enum.NotificationStatusQueued:
	case enum.NotificationStatusSent:
		if notification.SentAt == nil {
			notification.SentAt = &now
		}
		if strings.EqualFold(cmd.EventType, enum.ResendEventEmailDeliveryDelayed) {
			notification.LastError = formatResendWebhookFailure(cmd)
		} else if strings.EqualFold(cmd.EventType, enum.ResendEventEmailSent) {
			notification.LastError = ""
		}
	case enum.NotificationStatusDelivered:
		notification.LastError = ""
		if notification.SentAt == nil {
			notification.SentAt = &now
		}
		notification.DeliveredAt = &now
	case enum.NotificationStatusFailed:
		notification.LastError = formatResendWebhookFailure(cmd)
		notification.FailedAt = &now
	default:
	}
}

func shouldApplyProviderStatus(current, next enum.NotificationStatus) bool {
	return providerStatusRank(next) >= providerStatusRank(current)
}

func providerStatusRank(status enum.NotificationStatus) int {
	switch status {
	case enum.NotificationStatusFailed:
		return 50
	case enum.NotificationStatusDelivered:
		return 40
	case enum.NotificationStatusSent:
		return 30
	case enum.NotificationStatusProcessing:
		return 20
	case enum.NotificationStatusQueued, enum.NotificationStatusRetrying:
		return 10
	default:
		return 0
	}
}

func commandJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func stringFromJSON(value map[string]any, key string) string {
	v, ok := value[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func nestedStringFromJSON(value map[string]any, objectKey, key string) string {
	nested, ok := value[objectKey].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringFromJSON(nested, key))
}
