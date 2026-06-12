package godaddy

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	mail "github.com/wneessen/go-mail"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/infrastructure"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

type Provider struct {
	client  *mail.Client
	config  infrastructure.GoDaddyConfig
	initErr error
}

func NewProvider(config infrastructure.GoDaddyConfig) *Provider {
	client, err := newSMTPClient(config)
	return &Provider{client: client, config: config, initErr: err}
}

func newSMTPClient(config infrastructure.GoDaddyConfig) (*mail.Client, error) {
	if strings.TrimSpace(config.SMTPHost) == "" {
		return nil, apperror.ErrBadRequest(nil, "godaddy smtp host is required", apperror.KeyGodaddySMTPHostRequired)
	}
	port := config.SMTPPort
	if port == 0 {
		if config.UseImplicitTLS {
			port = 465
		} else {
			port = 587
		}
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	tlsConfig := &tls.Config{
		ServerName: config.SMTPHost,
		MinVersion: tls.VersionTLS12,
	}

	opts := []mail.Option{
		mail.WithPort(port),
		mail.WithUsername(config.Username),
		mail.WithPassword(config.Password),
		mail.WithTimeout(timeout),
		mail.WithTLSConfig(tlsConfig),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
	}
	if config.UseImplicitTLS {
		opts = append(opts, mail.WithSSLPort(true))
	} else {
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	}
	return mail.NewClient(config.SMTPHost, opts...)
}

func (p *Provider) Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error) {
	if p.initErr != nil {
		return command.ProviderSendResult{}, providerFailed(p.initErr, "godaddy smtp client is not configured")
	}

	msg, err := buildMessage(send, p.config)
	if err != nil {
		return command.ProviderSendResult{}, err
	}

	if err := p.client.DialAndSendWithContext(ctx, msg); err != nil {
		return command.ProviderSendResult{}, providerFailed(err, "godaddy smtp send failed")
	}

	return command.ProviderSendResult{
		MessageID: msg.GetMessageID(),
		Raw: types.JSONMap{
			"message_id":      msg.GetMessageID(),
			"server_response": msg.ServerResponse(),
			"from":            msg.GetFromString(),
			"to":              msg.GetToString(),
			"subject":         firstString(msg.GetGenHeader(mail.HeaderSubject)),
		},
	}, nil
}

func buildMessage(send command.ProviderSend, config infrastructure.GoDaddyConfig) (*mail.Msg, error) {
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
		return nil, apperror.ErrBadRequest(nil, "from address is required (metadata.from_address or GODADDY_FROM_ADDRESS)", apperror.KeyEmailFromRequired)
	}
	fromName := strings.TrimSpace(stringMeta(meta, enum.EmailMetaFromName))
	if fromName == "" {
		fromName = strings.TrimSpace(config.FromName)
	}

	msg := mail.NewMsg()
	from := fromAddress
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddress)
	}
	if err := msg.From(from); err != nil {
		return nil, apperror.ErrBadRequest(err, "invalid from address: "+err.Error(), apperror.KeyEmailInvalidFrom)
	}
	if err := msg.To(send.Recipient); err != nil {
		return nil, apperror.ErrBadRequest(err, "invalid recipient: "+err.Error(), apperror.KeyEmailInvalidRecipient)
	}
	if replyTo := strings.TrimSpace(stringMeta(meta, enum.EmailMetaReplyTo)); replyTo != "" {
		if err := msg.ReplyTo(replyTo); err != nil {
			return nil, apperror.ErrBadRequest(err, "invalid reply-to: "+err.Error(), apperror.KeyEmailInvalidReplyTo)
		}
	}
	if ccList := stringSliceMeta(meta, enum.EmailMetaCC); len(ccList) > 0 {
		if err := msg.Cc(ccList...); err != nil {
			return nil, apperror.ErrBadRequest(err, "invalid cc: "+err.Error(), apperror.KeyEmailInvalidCC)
		}
	}
	if bccList := stringSliceMeta(meta, enum.EmailMetaBCC); len(bccList) > 0 {
		if err := msg.Bcc(bccList...); err != nil {
			return nil, apperror.ErrBadRequest(err, "invalid bcc: "+err.Error(), apperror.KeyEmailInvalidBCC)
		}
	}

	msg.Subject(subject)
	msg.SetDate()
	msg.SetMessageID()

	contentType := strings.TrimSpace(stringMeta(meta, enum.EmailMetaContentType))
	if contentType == "" {
		contentType = enum.EmailContentTypePlain
	}
	switch contentType {
	case enum.EmailContentTypePlain:
		msg.SetBodyString(mail.TypeTextPlain, send.Body)
	case enum.EmailContentTypeHTML:
		msg.SetBodyString(mail.TypeTextHTML, send.Body)
	default:
		return nil, apperror.ErrBadRequest(nil, "metadata.content_type must be text/plain or text/html", apperror.KeyEmailContentTypeInvalid)
	}

	return msg, nil
}

func providerFailed(cause error, message string) *apperror.Error {
	return apperror.ErrBadRequestCode(cause, message, apperror.CodeProviderFailed, apperror.KeyProviderSendFailed)
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
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
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

// BuildMessageForTest exposes message construction for unit tests.
func BuildMessageForTest(send command.ProviderSend, config infrastructure.GoDaddyConfig) (*mail.Msg, error) {
	return buildMessage(send, config)
}
