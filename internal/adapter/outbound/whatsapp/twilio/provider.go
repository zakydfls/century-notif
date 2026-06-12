package twilio

import (
	"context"
	"encoding/json"
	"strings"

	twilioclient "github.com/twilio/twilio-go"
	twilioapi "github.com/twilio/twilio-go/rest/api/v2010"
	"notif-gateway/internal/value_object/util/type"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/infrastructure"
	apperror "notif-gateway/internal/value_object/handling/error"
)

const WhatsAppPrefix = "whatsapp:"

type Provider struct {
	client *twilioclient.RestClient
	config infrastructure.TwilioConfig
}

func NewProvider(config infrastructure.TwilioConfig) *Provider {
	client := twilioclient.NewRestClientWithParams(twilioclient.ClientParams{
		Username: config.AccountSID,
		Password: config.AuthToken,
	})
	return &Provider{client: client, config: config}
}

func (p *Provider) Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error) {
	params := &twilioapi.CreateMessageParams{}
	params.SetTo(ensureWhatsAppAddress(send.Recipient))
	if send.ProviderTemplateID != "" {
		params.SetContentSid(send.ProviderTemplateID)
		vars, err := json.Marshal(send.ProviderTemplateVars)
		if err != nil {
			return command.ProviderSendResult{}, err
		}
		params.SetContentVariables(string(vars))
	} else {
		params.SetBody(send.Body)
	}
	if p.config.WhatsAppService != "" {
		params.SetMessagingServiceSid(p.config.WhatsAppService)
	} else if p.config.WhatsAppFrom != "" {
		params.SetFrom(ensureWhatsAppAddress(p.config.WhatsAppFrom))
	} else {
		return command.ProviderSendResult{}, apperror.ErrBadRequest(nil, "twilio whatsapp sender or messaging service is required", apperror.KeyTwilioWhatsAppSenderRequired)
	}
	if base := strings.TrimSpace(p.config.WebhookBaseURL); base != "" {
		params.SetStatusCallback(strings.TrimRight(base, "/") + enum.TwilioWebhookPathWhatsApp)
	}
	response, err := p.client.Api.CreateMessage(params)
	if err != nil {
		return command.ProviderSendResult{}, apperror.ErrBadRequestCode(err, err.Error(), apperror.CodeProviderFailed, apperror.KeyProviderSendFailed)
	}
	return command.ProviderSendResult{
		MessageID: stringValue(response.Sid),
		Raw: types.JSONMap{
			"sid":    stringValue(response.Sid),
			"status": stringValue(response.Status),
		},
	}, nil
}

func ensureWhatsAppAddress(value string) string {
	if strings.HasPrefix(value, WhatsAppPrefix) {
		return value
	}
	return WhatsAppPrefix + value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
