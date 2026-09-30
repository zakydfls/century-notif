package whatsockt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/infrastructure"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

const (
	sendMessagePath   = "/api/v1/whatsockt/messages"
	maxResponseBytes  = 1 << 20
	maxErrorDetailLen = 512
)

type Provider struct {
	client *http.Client
	config infrastructure.WhatsocktConfig
}

func NewProvider(config infrastructure.WhatsocktConfig) *Provider {
	return &Provider{client: &http.Client{Timeout: config.Timeout}, config: config}
}

type sendMessageRequest struct {
	DeviceID  string         `json:"device_id"`
	Recipient string         `json:"recipient"`
	Content   messageContent `json:"content"`
}

type messageContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type sendMessageResponse struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	ProviderMessageID *string `json:"provider_message_id"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (p *Provider) Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(p.config.BaseURL), "/")
	apiKey := strings.TrimSpace(p.config.APIKey)
	deviceID := strings.TrimSpace(p.config.DeviceID)
	if baseURL == "" || apiKey == "" || deviceID == "" {
		return command.ProviderSendResult{}, apperror.ErrBadRequestCode(nil,
			"whatsockt base URL, API key, and device ID are required",
			apperror.CodeProviderFailed,
			apperror.KeyWhatsocktNotConfigured,
		)
	}
	if strings.TrimSpace(send.Body) == "" {
		return command.ProviderSendResult{}, apperror.ErrBadRequest(nil,
			"whatsockt text message body is required",
			apperror.KeyWhatsocktTextRequired,
		)
	}

	payload := sendMessageRequest{
		DeviceID:  deviceID,
		Recipient: normalizeRecipient(send.Recipient),
		Content:   messageContent{Type: "text", Text: send.Body},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return command.ProviderSendResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+sendMessagePath, bytes.NewReader(body))
	if err != nil {
		return command.ProviderSendResult{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", apiKey)
	if send.NotificationID != "" {
		request.Header.Set("Idempotency-Key", send.NotificationID)
	}

	response, err := p.client.Do(request)
	if err != nil {
		return command.ProviderSendResult{}, providerFailed(err, err.Error())
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if readErr != nil {
		return command.ProviderSendResult{}, providerFailed(readErr, "failed to read whatsockt response")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return command.ProviderSendResult{}, providerFailed(nil, formatError(response.StatusCode, responseBody))
	}

	var decoded sendMessageResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return command.ProviderSendResult{}, providerFailed(err, "invalid whatsockt response")
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return command.ProviderSendResult{}, providerFailed(errors.New("message id is missing"), "invalid whatsockt response")
	}
	return command.ProviderSendResult{
		MessageID: decoded.ID,
		Raw: types.JSONMap{
			"id":                  decoded.ID,
			"status":              decoded.Status,
			"provider_message_id": stringValue(decoded.ProviderMessageID),
			"status_code":         response.StatusCode,
		},
	}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizeRecipient(recipient string) string {
	recipient = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(recipient), "whatsapp:"))
	return strings.TrimSuffix(recipient, "@c.us")
}

func formatError(statusCode int, body []byte) string {
	message := fmt.Sprintf("whatsockt send failed (http %d)", statusCode)
	var decoded errorResponse
	if json.Unmarshal(body, &decoded) == nil {
		detail := strings.TrimSpace(strings.Join(nonEmpty(decoded.Error.Code, decoded.Error.Message), ": "))
		if detail != "" {
			return message + ": " + truncate(detail)
		}
	}
	if detail := strings.TrimSpace(string(body)); detail != "" {
		return message + ": " + truncate(detail)
	}
	return message
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func truncate(value string) string {
	if len(value) <= maxErrorDetailLen {
		return value
	}
	return value[:maxErrorDetailLen] + "..."
}

func providerFailed(cause error, message string) *apperror.Error {
	return apperror.ErrBadRequestCode(cause, message, apperror.CodeProviderFailed, apperror.KeyWhatsocktSendFailed)
}
