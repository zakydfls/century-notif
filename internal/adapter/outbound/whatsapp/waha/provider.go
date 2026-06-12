package waha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	types "notif-gateway/internal/value_object/util/type"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/infrastructure"
	apperror "notif-gateway/internal/value_object/handling/error"
)

const (
	HeaderAPIKey      = "X-Api-Key"
	SendTextPath      = "/api/sendText"
	maxErrorBodyBytes = 4096
	maxErrorDetailLen = 512
)

type Provider struct {
	client *http.Client
	config infrastructure.WAHAConfig
}

func NewProvider(config infrastructure.WAHAConfig) *Provider {
	return &Provider{
		client: &http.Client{Timeout: 10 * time.Second},
		config: config,
	}
}

type sendTextRequest struct {
	Session string `json:"session"`
	ChatID  string `json:"chatId"`
	Text    string `json:"text"`
}

type sendTextResponse struct {
	ID string `json:"id"`
}

type wahaErrorResponse struct {
	StatusCode int             `json:"statusCode"`
	Timestamp  string          `json:"timestamp"`
	Message    json.RawMessage `json:"message"`
	Error      string          `json:"error"`
	Exception  *wahaException  `json:"exception"`
}

type wahaException struct {
	Message string `json:"message"`
	Name    string `json:"name"`
	Stack   string `json:"stack"`
}

func (p *Provider) Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error) {
	payload := sendTextRequest{
		Session: p.config.Session,
		ChatID:  normalizeChatID(send.Recipient),
		Text:    send.Body,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return command.ProviderSendResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+SendTextPath, bytes.NewReader(body))
	if err != nil {
		return command.ProviderSendResult{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		request.Header.Set(HeaderAPIKey, p.config.APIKey)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return command.ProviderSendResult{}, apperror.ErrBadRequestCode(err, err.Error(), apperror.CodeProviderFailed, apperror.KeyProviderSendFailed)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return command.ProviderSendResult{}, apperror.ErrBadRequestCode(nil,
			formatSendTextError(response.StatusCode, payload.Session, payload.ChatID, responseBody),
			apperror.CodeProviderFailed,
			apperror.KeyWAHASendFailed,
		)
	}
	var decoded sendTextResponse
	_ = json.Unmarshal(responseBody, &decoded)
	return command.ProviderSendResult{
		MessageID: decoded.ID,
		Raw: types.JSONMap{
			"id":          decoded.ID,
			"status_code": response.StatusCode,
		},
	}, nil
}

func normalizeChatID(recipient string) string {
	recipient = strings.TrimSpace(recipient)
	if strings.Contains(recipient, "@") {
		local, domain, ok := strings.Cut(recipient, "@")
		if ok {
			return strings.TrimPrefix(local, "+") + "@" + domain
		}
		return recipient
	}
	return strings.TrimPrefix(recipient, "+") + "@c.us"
}

func formatSendTextError(statusCode int, session, chatID string, body []byte) string {
	msg := fmt.Sprintf("waha sendText failed (http %d, session=%s, chatId=%s)", statusCode, session, chatID)
	if detail := extractWAHAErrorDetail(body); detail != "" {
		return msg + ": " + detail
	}
	return msg
}

func extractWAHAErrorDetail(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}

	var parsed wahaErrorResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return truncateErrorDetail(trimmed)
	}

	if parts := collectWAHAErrorParts(parsed); len(parts) > 0 {
		return truncateErrorDetail(strings.Join(parts, " — "))
	}

	return wahaErrorFallback(parsed)
}

func collectWAHAErrorParts(parsed wahaErrorResponse) []string {
	parts := make([]string, 0, 2)

	if detail := formatJSONMessage(parsed.Message); isUsefulErrorText(detail) {
		parts = append(parts, detail)
	}

	if parsed.Exception != nil {
		if detail := formatExceptionDetail(parsed.Exception); isUsefulErrorText(detail) {
			parts = append(parts, detail)
		}
	}

	if parsed.Error != "" && !strings.EqualFold(parsed.Error, http.StatusText(parsed.StatusCode)) {
		parts = append(parts, parsed.Error)
	}

	return parts
}

func formatExceptionDetail(ex *wahaException) string {
	if ex == nil {
		return ""
	}
	if isUsefulErrorText(ex.Message) {
		if isUsefulErrorText(ex.Name) && ex.Name != ex.Message {
			return ex.Name + ": " + ex.Message
		}
		return ex.Message
	}
	return firstStackLine(ex.Stack)
}

func wahaErrorFallback(parsed wahaErrorResponse) string {
	if parsed.Timestamp != "" {
		return fmt.Sprintf("whatsapp engine error at %s (verify WAHA session is connected)", parsed.Timestamp)
	}
	return "whatsapp engine error (verify WAHA session is connected)"
}

func isUsefulErrorText(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) <= 2 {
		return false
	}
	if idx := strings.Index(value, ": "); idx > 0 && idx <= 3 {
		prefix := value[:idx]
		suffix := value[idx+2:]
		if prefix == suffix && len(prefix) <= 2 {
			return false
		}
	}
	return true
}

func firstStackLine(stack string) string {
	line, _, _ := strings.Cut(stack, "\n")
	return strings.TrimSpace(line)
}

func formatJSONMessage(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var message string
	if err := json.Unmarshal(raw, &message); err == nil {
		return strings.TrimSpace(message)
	}

	var messages []string
	if err := json.Unmarshal(raw, &messages); err == nil {
		return strings.TrimSpace(strings.Join(messages, "; "))
	}

	return strings.TrimSpace(string(raw))
}

func truncateErrorDetail(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxErrorDetailLen {
		return value
	}
	return value[:maxErrorDetailLen] + "..."
}
