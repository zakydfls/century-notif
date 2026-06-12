package alibaba

import (
	"context"
	"encoding/json"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20180501/v2/client"
	"notif-gateway/internal/value_object/util/type"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/infrastructure"
	apperror "notif-gateway/internal/value_object/handling/error"
)

type Provider struct {
	client  *dysmsapi.Client
	config  infrastructure.AlibabaConfig
	initErr error // set if dysmsapi.NewClient fails at construction
}

func NewProvider(config infrastructure.AlibabaConfig) *Provider {
	client, err := dysmsapi.NewClient(&openapi.Config{
		AccessKeyId:     &config.AccessKeyID,
		AccessKeySecret: &config.AccessKeySecret,
		Endpoint:        &config.Endpoint,
		RegionId:        &config.RegionID,
	})
	return &Provider{client: client, config: config, initErr: err}
}

func (p *Provider) Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error) {
	if p.initErr != nil {
		return command.ProviderSendResult{}, apperror.ErrBadRequestCode(p.initErr, "alibaba sms client is not configured", apperror.CodeProviderFailed, apperror.KeyAlibabaClientNotConfigured)
	}
	if p.config.Mode == enum.AlibabaModeChina {
		return p.sendChina(send)
	}
	return p.sendGlobe(send)
}

func (p *Provider) sendGlobe(send command.ProviderSend) (command.ProviderSendResult, error) {
	request := &dysmsapi.SendMessageToGlobeRequest{}
	request.SetTo(send.Recipient)
	request.SetMessage(send.Body)
	if p.config.From != "" {
		request.SetFrom(p.config.From)
	}
	if p.config.ChannelID != "" {
		request.SetChannelId(p.config.ChannelID)
	}
	response, err := p.client.SendMessageToGlobe(request)
	if err != nil {
		return command.ProviderSendResult{}, apperror.ErrBadRequestCode(err, err.Error(), apperror.CodeProviderFailed, apperror.KeyProviderSendFailed)
	}
	if response.Body == nil {
		return command.ProviderSendResult{}, nil
	}
	return command.ProviderSendResult{
		MessageID: stringValue(response.Body.MessageId),
		Raw: types.JSONMap{
			"message_id":    stringValue(response.Body.MessageId),
			"request_id":    stringValue(response.Body.RequestId),
			"response_code": stringValue(response.Body.ResponseCode),
		},
	}, nil
}

func (p *Provider) sendChina(send command.ProviderSend) (command.ProviderSendResult, error) {
	if send.ProviderTemplateID == "" {
		return command.ProviderSendResult{}, apperror.ErrBadRequest(nil, "alibaba china mode requires provider template id", apperror.KeyAlibabaTemplateRequired)
	}
	vars, err := json.Marshal(send.ProviderTemplateVars)
	if err != nil {
		return command.ProviderSendResult{}, err
	}
	request := &dysmsapi.SendMessageWithTemplateRequest{}
	request.SetTo(send.Recipient)
	request.SetTemplateCode(send.ProviderTemplateID)
	request.SetTemplateParam(string(vars))
	if p.config.From != "" {
		request.SetFrom(p.config.From)
	}
	if p.config.ChannelID != "" {
		request.SetChannelId(p.config.ChannelID)
	}
	response, err := p.client.SendMessageWithTemplate(request)
	if err != nil {
		return command.ProviderSendResult{}, apperror.ErrBadRequestCode(err, err.Error(), apperror.CodeProviderFailed, apperror.KeyProviderSendFailed)
	}
	if response.Body == nil {
		return command.ProviderSendResult{}, nil
	}
	return command.ProviderSendResult{
		MessageID: stringValue(response.Body.MessageId),
		Raw: types.JSONMap{
			"message_id":    stringValue(response.Body.MessageId),
			"request_id":    stringValue(response.Body.RequestId),
			"response_code": stringValue(response.Body.ResponseCode),
		},
	}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
