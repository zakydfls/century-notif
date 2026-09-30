package rest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	svix "github.com/svix/svix-webhooks/go"
	twilioClient "github.com/twilio/twilio-go/client"

	"github.com/go-chi/chi/v5"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/service"
	"notif-gateway/internal/pkg/validator"
	"notif-gateway/internal/value_object/dto/response"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

const maxTwilioWebhookBodyBytes = 1 << 20    // 1 MiB
const maxWAHAWebhookBodyBytes = 1 << 20      // 1 MiB
const maxWhatsocktWebhookBodyBytes = 1 << 20 // 1 MiB
const maxResendWebhookBodyBytes = 1 << 20    // 1 MiB
const whatsocktWebhookTolerance = 5 * time.Minute

type WebhookREST interface {
	RouteModule

	TwilioSMS(w http.ResponseWriter, r *http.Request) error
	TwilioWhatsApp(w http.ResponseWriter, r *http.Request) error
	WAHAWhatsApp(w http.ResponseWriter, r *http.Request) error
	WhatsocktWhatsApp(w http.ResponseWriter, r *http.Request) error
	Resend(w http.ResponseWriter, r *http.Request) error
}

func NewWebhookREST(webhooks service.Webhook, twilio TwilioWebhook, waha WAHAWebhook, whatsockt WhatsocktWebhook, resend ResendWebhook) WebhookREST {
	validateEffective := twilio.ValidateSignature && twilio.AuthToken != ""
	return &webhookREST{
		webhooks:                webhooks,
		twilioWebhookCfg:        twilio,
		twilioValidateEffective: validateEffective,
		wahaWebhookCfg:          waha,
		whatsocktWebhookCfg:     whatsockt,
		resendWebhookCfg:        resend,
	}
}

type webhookREST struct {
	webhooks                service.Webhook
	twilioWebhookCfg        TwilioWebhook
	twilioValidateEffective bool
	wahaWebhookCfg          WAHAWebhook
	whatsocktWebhookCfg     WhatsocktWebhook
	resendWebhookCfg        ResendWebhook
}

func (h *webhookREST) GetRoutePath() string {
	return "/webhooks"
}

func (h *webhookREST) Start(_ context.Context) chi.Router {
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

func (h *webhookREST) Mount(r chi.Router) {
	r.Post("/twilio/sms", RootHandler(h.TwilioSMS))
	r.Post("/twilio/whatsapp", RootHandler(h.TwilioWhatsApp))
	r.Post("/waha/whatsapp", RootHandler(h.WAHAWhatsApp))
	r.Post("/whatsockt/whatsapp", RootHandler(h.WhatsocktWhatsApp))
	r.Post("/resend", RootHandler(h.Resend))
}

func (h *webhookREST) TwilioSMS(w http.ResponseWriter, r *http.Request) error {
	return h.twilioWebhook(w, r, enum.ChannelSMS)
}

func (h *webhookREST) TwilioWhatsApp(w http.ResponseWriter, r *http.Request) error {
	return h.twilioWebhook(w, r, enum.ChannelWhatsApp)
}

func (h *webhookREST) WAHAWhatsApp(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWAHAWebhookBodyBytes))
	if err != nil {
		return apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookBodyReadFailed)
	}
	if err := h.verifyWAHAWebhook(body, r.Header); err != nil {
		return err
	}
	cmd, err := wahaWebhookCommand(body, r.Header)
	if err != nil {
		return err
	}
	if err := h.webhooks.HandleWAHA(r.Context(), cmd); err != nil {
		return err
	}
	response.SuccessData(w, "webhook received", http.StatusOK, map[string]string{"provider": enum.ProviderWAHA.String()})
	return nil
}

func (h *webhookREST) verifyWAHAWebhook(body []byte, header http.Header) error {
	if !h.wahaWebhookCfg.ValidateSignature {
		return nil
	}
	key := strings.TrimSpace(h.wahaWebhookCfg.HMACKey)
	if key == "" {
		return apperror.ErrForbidden(nil, "waha webhook hmac key is required", apperror.KeyWAHAWebhookHMACKeyRequired)
	}
	algorithm := strings.TrimSpace(strings.ToLower(header.Get("X-Webhook-Hmac-Algorithm")))
	if algorithm != "" && algorithm != "sha512" {
		return apperror.ErrForbidden(nil, "unsupported waha webhook hmac algorithm", apperror.KeyWebhookInvalidSign)
	}
	signature := strings.TrimSpace(header.Get("X-Webhook-Hmac"))
	if signature == "" {
		return apperror.ErrForbidden(nil, "missing waha webhook hmac", apperror.KeyWebhookInvalidSign)
	}
	expected := wahaWebhookHMAC(body, key)
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(signature)), []byte(expected)) != 1 {
		return apperror.ErrForbidden(nil, "invalid waha webhook hmac", apperror.KeyWebhookInvalidSign)
	}
	return nil
}

func (h *webhookREST) WhatsocktWhatsApp(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWhatsocktWebhookBodyBytes))
	if err != nil {
		return apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookBodyReadFailed)
	}
	if err := h.verifyWhatsocktWebhook(body, r.Header); err != nil {
		return err
	}
	cmd, err := whatsocktWebhookCommand(body, r.Header)
	if err != nil {
		return err
	}
	if err := h.webhooks.HandleWhatsockt(r.Context(), cmd); err != nil {
		return err
	}
	response.SuccessData(w, "webhook received", http.StatusOK, map[string]string{"provider": enum.ProviderWhatsockt.String()})
	return nil
}

func (h *webhookREST) verifyWhatsocktWebhook(body []byte, header http.Header) error {
	if !h.whatsocktWebhookCfg.ValidateSignature {
		return nil
	}
	secret := strings.TrimSpace(h.whatsocktWebhookCfg.Secret)
	if secret == "" {
		return apperror.ErrForbidden(nil, "whatsockt webhook secret is required", apperror.KeyWhatsocktWebhookKeyRequired)
	}
	timestamp := strings.TrimSpace(header.Get("X-WA-Timestamp"))
	sentAt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return apperror.ErrForbidden(err, "invalid whatsockt webhook timestamp", apperror.KeyWebhookInvalidSign)
	}
	delta := time.Now().Unix() - sentAt
	if delta < -int64(whatsocktWebhookTolerance/time.Second) || delta > int64(whatsocktWebhookTolerance/time.Second) {
		return apperror.ErrForbidden(nil, "expired whatsockt webhook timestamp", apperror.KeyWebhookInvalidSign)
	}
	signature := strings.TrimSpace(header.Get("X-WA-Signature"))
	expected := whatsocktWebhookHMAC(timestamp, body, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return apperror.ErrForbidden(nil, "invalid whatsockt webhook signature", apperror.KeyWebhookInvalidSign)
	}
	return nil
}

func (h *webhookREST) Resend(w http.ResponseWriter, r *http.Request) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxResendWebhookBodyBytes))
	if err != nil {
		return apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookBodyReadFailed)
	}
	if err := h.verifyResendWebhook(body, r.Header); err != nil {
		return err
	}
	cmd, err := resendWebhookCommand(body, r.Header)
	if err != nil {
		return err
	}
	if err := h.webhooks.HandleResend(r.Context(), cmd); err != nil {
		return err
	}
	response.SuccessData(w, "webhook received", http.StatusOK, map[string]string{"provider": enum.ProviderResend.String()})
	return nil
}

func (h *webhookREST) verifyResendWebhook(body []byte, header http.Header) error {
	if !h.resendWebhookCfg.ValidateSignature {
		return nil
	}
	secret := strings.TrimSpace(h.resendWebhookCfg.Secret)
	if secret == "" {
		return apperror.ErrForbidden(nil, "resend webhook secret is required", apperror.KeyResendWebhookRequired)
	}
	wh, err := svix.NewWebhook(secret)
	if err != nil {
		return apperror.ErrForbidden(err, "invalid resend webhook secret", apperror.KeyResendWebhookRequired)
	}
	if err := wh.Verify(body, header); err != nil {
		return apperror.ErrForbidden(err, "invalid resend webhook signature", apperror.KeyWebhookInvalidSign)
	}
	return nil
}

func (h *webhookREST) twilioWebhook(w http.ResponseWriter, r *http.Request, channel enum.Channel) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxTwilioWebhookBodyBytes))
	if err != nil {
		return apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookBodyReadFailed)
	}

	path := enum.TwilioWebhookPathSMS
	if channel == enum.ChannelWhatsApp {
		path = enum.TwilioWebhookPathWhatsApp
	}

	if h.twilioValidateEffective {
		sig := strings.TrimSpace(r.Header.Get("X-Twilio-Signature"))
		if sig == "" {
			return apperror.ErrForbidden(nil, "", apperror.KeyForbidden)
		}
		validatorURL := twilioWebhookCanonicalURL(strings.TrimSpace(h.twilioWebhookCfg.BaseURL), path, r)
		rv := twilioClient.NewRequestValidator(h.twilioWebhookCfg.AuthToken)
		if !rv.ValidateBody(validatorURL, body, sig) {
			return apperror.ErrForbidden(nil, "", apperror.KeyForbidden)
		}
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookInvalidForm)
	}

	status := values.Get("MessageStatus")
	if status == "" {
		status = values.Get("SmsStatus")
	}
	if err := h.webhooks.Handle(r.Context(), command.HandleWebhook{
		Provider:             enum.ProviderTwilio,
		ProviderMessageID:    values.Get("MessageSid"),
		ExternalStatus:       status,
		ErrorCode:            values.Get("ErrorCode"),
		ErrorMessage:         values.Get("ErrorMessage"),
		ChannelStatusMessage: values.Get("ChannelStatusMessage"),
	}); err != nil {
		return err
	}
	response.SuccessData(w, "webhook received", http.StatusOK, map[string]string{"channel": channel.String()})
	return nil
}

func wahaWebhookCommand(body []byte, header http.Header) (command.HandleWAHAWebhook, error) {
	var payload types.JSONMap
	if err := json.Unmarshal(body, &payload); err != nil {
		return command.HandleWAHAWebhook{}, apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookInvalidJSON)
	}

	eventType := strings.TrimSpace(stringFromAny(payload["event"]))
	if eventType == "" {
		return command.HandleWAHAWebhook{}, apperror.ErrBadRequest(nil, "waha webhook event is required", apperror.KeyWebhookInvalidJSON)
	}

	var eventCreatedAt time.Time
	if rawTimestamp, ok := int64FromAny(payload["timestamp"]); ok && rawTimestamp > 0 {
		eventCreatedAt = unixMillis(rawTimestamp)
	}

	data := jsonMapFromAny(payload["payload"])
	eventID := strings.TrimSpace(stringFromAny(payload["id"]))
	if eventID == "" {
		eventID = strings.TrimSpace(header.Get("X-Webhook-Request-Id"))
	}
	if eventID == "" {
		eventID = fallbackResendEventID(body)
	}

	ack, ackPresent := intFromAny(data["ack"])
	return command.HandleWAHAWebhook{
		EventID:           eventID,
		EventType:         eventType,
		EventCreatedAt:    eventCreatedAt,
		Session:           strings.TrimSpace(stringFromAny(payload["session"])),
		ProviderMessageID: strings.TrimSpace(stringFromAny(data["id"])),
		AckName:           strings.TrimSpace(stringFromAny(data["ackName"])),
		Ack:               ack,
		AckPresent:        ackPresent,
		Payload:           payload,
		Data:              data,
		Headers: types.JSONMap{
			"x_webhook_request_id":     strings.TrimSpace(header.Get("X-Webhook-Request-Id")),
			"x_webhook_timestamp":      strings.TrimSpace(header.Get("X-Webhook-Timestamp")),
			"x_webhook_hmac_algorithm": strings.TrimSpace(header.Get("X-Webhook-Hmac-Algorithm")),
			"x_webhook_hmac":           strings.TrimSpace(header.Get("X-Webhook-Hmac")),
		},
	}, nil
}

func whatsocktWebhookCommand(body []byte, header http.Header) (command.HandleWhatsocktWebhook, error) {
	var payload types.JSONMap
	if err := json.Unmarshal(body, &payload); err != nil {
		return command.HandleWhatsocktWebhook{}, apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookInvalidJSON)
	}
	eventType := strings.TrimSpace(stringFromAny(payload["type"]))
	if eventType == "" {
		return command.HandleWhatsocktWebhook{}, apperror.ErrBadRequest(nil, "whatsockt webhook type is required", apperror.KeyWebhookInvalidJSON)
	}
	var eventCreatedAt time.Time
	if raw := strings.TrimSpace(stringFromAny(payload["occurred_at"])); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return command.HandleWhatsocktWebhook{}, apperror.ErrBadRequest(err, "invalid whatsockt webhook occurred_at", apperror.KeyWebhookInvalidJSON)
		}
		eventCreatedAt = parsed
	}
	data := jsonMapFromAny(payload["data"])
	eventID := strings.TrimSpace(header.Get("X-WA-Delivery-ID"))
	if eventID == "" {
		eventID = strings.TrimSpace(stringFromAny(payload["id"]))
	}
	if eventID == "" {
		eventID = fallbackResendEventID(body)
	}
	return command.HandleWhatsocktWebhook{
		EventID:           eventID,
		EventType:         eventType,
		EventCreatedAt:    eventCreatedAt,
		ProviderMessageID: strings.TrimSpace(stringFromAny(data["message_id"])),
		ErrorCode:         strings.TrimSpace(stringFromAny(data["error_code"])),
		ErrorMessage:      strings.TrimSpace(stringFromAny(data["error_message"])),
		Payload:           payload,
		Headers: types.JSONMap{
			"x_wa_timestamp":   strings.TrimSpace(header.Get("X-WA-Timestamp")),
			"x_wa_signature":   strings.TrimSpace(header.Get("X-WA-Signature")),
			"x_wa_delivery_id": strings.TrimSpace(header.Get("X-WA-Delivery-ID")),
		},
	}, nil
}

func resendWebhookCommand(body []byte, header http.Header) (command.HandleResendWebhook, error) {
	var payload types.JSONMap
	if err := json.Unmarshal(body, &payload); err != nil {
		return command.HandleResendWebhook{}, apperror.ErrBadRequest(err, err.Error(), apperror.KeyWebhookInvalidJSON)
	}

	eventType := strings.TrimSpace(stringFromAny(payload["type"]))
	if eventType == "" {
		return command.HandleResendWebhook{}, apperror.ErrBadRequest(nil, "resend webhook type is required", apperror.KeyWebhookInvalidJSON)
	}

	var eventCreatedAt time.Time
	if rawCreatedAt := strings.TrimSpace(stringFromAny(payload["created_at"])); rawCreatedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, rawCreatedAt)
		if err != nil {
			return command.HandleResendWebhook{}, apperror.ErrBadRequest(err, "invalid resend webhook created_at", apperror.KeyWebhookInvalidJSON)
		}
		eventCreatedAt = parsed
	}

	data := jsonMapFromAny(payload["data"])
	eventID := strings.TrimSpace(header.Get("svix-id"))
	if eventID == "" {
		eventID = fallbackResendEventID(body)
	}

	return command.HandleResendWebhook{
		EventID:           eventID,
		EventType:         eventType,
		EventCreatedAt:    eventCreatedAt,
		ProviderMessageID: strings.TrimSpace(stringFromAny(data["email_id"])),
		Data:              data,
		Payload:           payload,
		Headers: types.JSONMap{
			"svix_id":        strings.TrimSpace(header.Get("svix-id")),
			"svix_timestamp": strings.TrimSpace(header.Get("svix-timestamp")),
			"svix_signature": strings.TrimSpace(header.Get("svix-signature")),
		},
	}, nil
}

func (h *webhookREST) jsonWebhook(w http.ResponseWriter, r *http.Request, provider enum.Provider) error {
	var request command.WebhookInbound
	if err := validator.Bind(r, &request); err != nil {
		return err
	}
	if err := h.webhooks.Handle(r.Context(), request.ToHandleWebhook(provider)); err != nil {
		return err
	}
	response.SuccessData(w, "webhook received", http.StatusOK, map[string]string{"status": "ok"})
	return nil
}

func fallbackResendEventID(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func wahaWebhookHMAC(body []byte, key string) string {
	mac := hmac.New(sha512.New, []byte(key))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func whatsocktWebhookHMAC(timestamp string, body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func jsonMapFromAny(value any) types.JSONMap {
	switch typed := value.(type) {
	case map[string]any:
		return types.JSONMap(typed)
	case types.JSONMap:
		return typed
	default:
		return types.JSONMap{}
	}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		i, err := typed.Int64()
		if err == nil {
			return int(i), true
		}
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

func int64FromAny(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		i, err := typed.Int64()
		if err == nil {
			return i, true
		}
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

func unixMillis(value int64) time.Time {
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value)
	}
	return time.Unix(value, 0)
}

// twilioWebhookCanonicalURL is the absolute URL Twilio signs and POSTs to.
func twilioWebhookCanonicalURL(trimmedConfiguredBaseURL, webhookPath string, r *http.Request) string {
	if trimmedConfiguredBaseURL != "" {
		return strings.TrimRight(trimmedConfiguredBaseURL, "/") + webhookPath
	}
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	if fwd := csvFirstTrimmed(r.Header.Get("X-Forwarded-Proto")); fwd != "" {
		proto = fwd
	}
	host := csvFirstTrimmed(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return proto + "://" + host + r.URL.Path
}

func csvFirstTrimmed(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	idx := strings.IndexByte(header, ',')
	if idx < 0 {
		return strings.TrimSpace(header)
	}
	return strings.TrimSpace(header[:idx])
}
