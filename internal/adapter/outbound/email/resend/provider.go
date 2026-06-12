package resend

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	stdmail "net/mail"
	"strings"
	"time"

	resendgo "github.com/resend/resend-go/v3"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/infrastructure"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

type emailSender interface {
	SendWithContext(ctx context.Context, params *resendgo.SendEmailRequest) (*resendgo.SendEmailResponse, error)
}

type Provider struct {
	emails  emailSender
	config  infrastructure.ResendConfig
	initErr error
}

func NewProvider(config infrastructure.ResendConfig) *Provider {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return &Provider{
			config:  config,
			initErr: apperror.ErrBadRequest(nil, "resend api key is required", apperror.KeyResendAPIKeyRequired),
		}
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client := resendgo.NewCustomClient(&http.Client{Timeout: timeout}, apiKey)
	return &Provider{emails: client.Emails, config: config}
}

func (p *Provider) Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error) {
	if p.initErr != nil {
		return command.ProviderSendResult{}, providerFailed(p.initErr, "resend client is not configured")
	}
	if p.emails == nil {
		return command.ProviderSendResult{}, providerFailed(errors.New("resend emails client is nil"), "resend client is not configured")
	}

	request, err := buildRequest(send, p.config)
	if err != nil {
		return command.ProviderSendResult{}, err
	}

	response, err := p.emails.SendWithContext(ctx, request)
	if err != nil {
		return command.ProviderSendResult{}, providerFailed(err, "resend email send failed")
	}

	messageID := ""
	if response != nil {
		messageID = response.Id
	}
	raw := types.JSONMap{
		"id":      messageID,
		"from":    request.From,
		"to":      request.To,
		"subject": request.Subject,
	}
	if request.Html != "" {
		raw["content_type"] = enum.EmailContentTypeHTML
	} else if request.Text != "" {
		raw["content_type"] = enum.EmailContentTypePlain
	}
	if len(request.Cc) > 0 {
		raw["cc"] = request.Cc
	}
	if len(request.Bcc) > 0 {
		raw["bcc"] = request.Bcc
	}
	if request.ReplyTo != "" {
		raw["reply_to"] = request.ReplyTo
	}
	if request.Template != nil {
		raw["template_id"] = request.Template.Id
	}

	return command.ProviderSendResult{
		MessageID: messageID,
		Raw:       raw,
	}, nil
}

func buildRequest(send command.ProviderSend, config infrastructure.ResendConfig) (*resendgo.SendEmailRequest, error) {
	meta := send.Metadata
	if meta == nil {
		meta = types.JSONMap{}
	}

	subject := strings.TrimSpace(stringMeta(meta, enum.EmailMetaSubject))
	if subject == "" {
		return nil, apperror.ErrBadRequest(nil, "email subject is required in metadata", apperror.KeyEmailSubjectRequired)
	}

	fromAddress := strings.TrimSpace(stringMeta(meta, enum.EmailMetaFromAddress))
	if fromAddress == "" {
		fromAddress = strings.TrimSpace(config.FromAddress)
	}
	if fromAddress == "" {
		return nil, apperror.ErrBadRequest(nil, "from address is required (metadata.from_address or RESEND_FROM_ADDRESS)", apperror.KeyEmailFromRequired)
	}
	fromName := strings.TrimSpace(stringMeta(meta, enum.EmailMetaFromName))
	if fromName == "" {
		fromName = strings.TrimSpace(config.FromName)
	}

	from, err := formatAddress(fromAddress, fromName)
	if err != nil {
		return nil, apperror.ErrBadRequest(err, "invalid from address: "+err.Error(), apperror.KeyEmailInvalidFrom)
	}

	recipient := strings.TrimSpace(send.Recipient)
	if err := validateAddress(recipient); err != nil {
		return nil, apperror.ErrBadRequest(err, "invalid recipient: "+err.Error(), apperror.KeyEmailInvalidRecipient)
	}

	request := &resendgo.SendEmailRequest{
		From:    from,
		To:      []string{recipient},
		Subject: subject,
	}

	if replyTo := strings.TrimSpace(stringMeta(meta, enum.EmailMetaReplyTo)); replyTo != "" {
		if err := validateAddress(replyTo); err != nil {
			return nil, apperror.ErrBadRequest(err, "invalid reply-to: "+err.Error(), apperror.KeyEmailInvalidReplyTo)
		}
		request.ReplyTo = replyTo
	}

	ccList := stringSliceMeta(meta, enum.EmailMetaCC)
	if err := validateAddressList(ccList); err != nil {
		return nil, apperror.ErrBadRequest(err, "invalid cc: "+err.Error(), apperror.KeyEmailInvalidCC)
	}
	request.Cc = ccList

	bccList := stringSliceMeta(meta, enum.EmailMetaBCC)
	if err := validateAddressList(bccList); err != nil {
		return nil, apperror.ErrBadRequest(err, "invalid bcc: "+err.Error(), apperror.KeyEmailInvalidBCC)
	}
	request.Bcc = bccList

	if templateID := strings.TrimSpace(send.ProviderTemplateID); templateID != "" {
		request.Template = &resendgo.EmailTemplate{
			Id:        templateID,
			Variables: map[string]any(send.ProviderTemplateVars),
		}
		return request, nil
	}

	contentType := strings.TrimSpace(stringMeta(meta, enum.EmailMetaContentType))
	if contentType == "" {
		contentType = enum.EmailContentTypePlain
	}
	switch contentType {
	case enum.EmailContentTypePlain:
		request.Text = send.Body
	case enum.EmailContentTypeHTML:
		request.Html = send.Body
	default:
		return nil, apperror.ErrBadRequest(nil, "metadata.content_type must be text/plain or text/html", apperror.KeyEmailContentTypeInvalid)
	}

	return request, nil
}

func providerFailed(cause error, message string) *apperror.Error {
	return apperror.ErrBadRequestCode(cause, message, apperror.CodeProviderFailed, apperror.KeyProviderSendFailed)
}

func formatAddress(address string, name string) (string, error) {
	value := strings.TrimSpace(address)
	if name != "" {
		value = (&stdmail.Address{Name: name, Address: value}).String()
	}
	if err := validateAddress(value); err != nil {
		return "", err
	}
	return value, nil
}

func validateAddress(value string) error {
	_, err := stdmail.ParseAddress(strings.TrimSpace(value))
	return err
}

func validateAddressList(values []string) error {
	for _, value := range values {
		if err := validateAddress(value); err != nil {
			return fmt.Errorf("%s: %w", value, err)
		}
	}
	return nil
}

func stringMeta(meta types.JSONMap, key string) string {
	v, ok := meta[key]
	if !ok || v == nil {
		return ""
	}
	switch typed := v.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func stringSliceMeta(meta types.JSONMap, key string) []string {
	v, ok := meta[key]
	if !ok || v == nil {
		return nil
	}
	switch typed := v.(type) {
	case []string:
		return compactStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return compactStrings(out)
	default:
		return nil
	}
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

// BuildRequestForTest exposes Resend request construction for unit tests.
func BuildRequestForTest(send command.ProviderSend, config infrastructure.ResendConfig) (*resendgo.SendEmailRequest, error) {
	return buildRequest(send, config)
}
