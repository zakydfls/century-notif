package service

import (
	"context"
	"net/mail"
	"strings"

	disposable "github.com/mehrtat/disposable-email-domain"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

type NotificationBuilder struct {
	renderer             *TemplateRenderer
	dispatcher           *Dispatcher
	blockDisposableEmail bool
	emailAllowlist       map[string]struct{}
}

type EmailSecurityPolicy struct {
	BlockDisposable     bool
	DisposableAllowlist []string
}

func NewNotificationBuilder(renderer *TemplateRenderer, dispatcher *Dispatcher, policy EmailSecurityPolicy) *NotificationBuilder {
	allowlist := make(map[string]struct{}, len(policy.DisposableAllowlist))
	for _, domain := range policy.DisposableAllowlist {
		if domain = strings.ToLower(strings.TrimSpace(domain)); domain != "" {
			allowlist[domain] = struct{}{}
		}
	}
	return &NotificationBuilder{
		renderer:             renderer,
		dispatcher:           dispatcher,
		blockDisposableEmail: policy.BlockDisposable,
		emailAllowlist:       allowlist,
	}
}

func (b *NotificationBuilder) Build(ctx context.Context, input command.Enqueue) ([]*entity.Notification, error) {
	serviceName := strings.TrimSpace(input.ServiceName)
	if len(serviceName) > 128 {
		return nil, apperror.ErrBadRequest(nil, "service_name must be at most 128 characters", apperror.KeyServiceNameInvalid)
	}
	if !input.Channel.IsValid() {
		return nil, apperror.ErrBadRequest(nil, "invalid notification channel", apperror.KeyInvalidChannel)
	}
	if input.Body == "" && input.TemplateKey == "" && input.ProviderTemplateID == "" {
		return nil, apperror.ErrBadRequest(nil, "body, template key, or provider template id is required", apperror.KeyMissingContent)
	}
	if input.Channel == enum.ChannelEmail {
		if err := validateEmailMetadata(input.Metadata); err != nil {
			return nil, err
		}
		for _, recipient := range input.Recipients {
			if err := b.validateEmailRecipient(recipient); err != nil {
				return nil, err
			}
		}
		for _, recipient := range emailMetadataRecipients(input.Metadata) {
			if err := b.validateEmailRecipient(recipient); err != nil {
				return nil, err
			}
		}
	}

	provider := b.dispatcher.DefaultProvider(input.Channel)
	var renderResult command.RenderTemplateResult
	if input.TemplateKey != "" {
		var err error
		renderResult, err = b.renderer.Render(ctx, command.RenderTemplate{
			Key:      input.TemplateKey,
			Channel:  input.Channel,
			Provider: provider,
			Params:   NormalizeJSONMap(input.TemplateParams),
		})
		if err != nil {
			return nil, err
		}
	}

	body := input.Body
	if body == "" {
		body = renderResult.Body
	}
	providerTemplateID := input.ProviderTemplateID
	if providerTemplateID == "" {
		providerTemplateID = renderResult.ProviderTemplateID
	}

	notifications := make([]*entity.Notification, 0, len(input.Recipients))
	for _, recipient := range input.Recipients {
		notifications = append(notifications, &entity.Notification{
			UserID:               input.UserID,
			ServiceName:          serviceName,
			Channel:              input.Channel,
			Provider:             provider,
			Recipient:            recipient,
			Body:                 body,
			TemplateKey:          input.TemplateKey,
			TemplateParams:       NormalizeJSONMap(input.TemplateParams),
			ProviderTemplateID:   providerTemplateID,
			ProviderTemplateVars: NormalizeJSONMap(input.ProviderTemplateVars),
			Metadata:             NormalizeJSONMap(input.Metadata),
			Status:               enum.NotificationStatusQueued,
		})
	}
	return notifications, nil
}

func (b *NotificationBuilder) validateEmailRecipient(recipient string) error {
	address, err := mail.ParseAddress(strings.TrimSpace(recipient))
	if err != nil {
		return apperror.ErrBadRequest(err, "invalid email recipient", apperror.KeyEmailInvalidRecipient)
	}
	if !b.blockDisposableEmail {
		return nil
	}
	domain := emailDomain(address.Address)
	if domainOrParentInSet(domain, func(candidate string) bool {
		_, allowed := b.emailAllowlist[candidate]
		return allowed
	}) {
		return nil
	}
	if domainOrParentInSet(domain, disposable.IsDisposable) {
		return apperror.ErrBadRequest(nil, "disposable email addresses are not allowed", apperror.KeyEmailDisposableRecipient)
	}
	return nil
}

func domainOrParentInSet(domain string, contains func(string) bool) bool {
	for strings.Contains(domain, ".") {
		if contains(domain) {
			return true
		}
		_, domain, _ = strings.Cut(domain, ".")
	}
	return false
}

func emailMetadataRecipients(metadata types.JSONMap) []string {
	var result []string
	for _, key := range []string{enum.EmailMetaCC, enum.EmailMetaBCC} {
		switch values := metadata[key].(type) {
		case []string:
			result = append(result, values...)
		case []any:
			for _, value := range values {
				if recipient, ok := value.(string); ok {
					result = append(result, recipient)
				}
			}
		}
	}
	return result
}

func emailDomain(address string) string {
	address = strings.ToLower(strings.TrimSpace(address))
	if at := strings.LastIndexByte(address, '@'); at >= 0 {
		return strings.TrimRight(address[at+1:], ".")
	}
	return ""
}

func NormalizeJSONMap(value types.JSONMap) types.JSONMap {
	if value == nil {
		return types.JSONMap{}
	}
	return value
}

func validateEmailMetadata(metadata types.JSONMap) error {
	subject, _ := metadata[enum.EmailMetaSubject].(string)
	if strings.TrimSpace(subject) == "" {
		return apperror.ErrBadRequest(nil, "metadata.subject is required for email channel", apperror.KeyEmailSubjectRequired)
	}
	contentType, _ := metadata[enum.EmailMetaContentType].(string)
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return nil
	}
	if contentType != enum.EmailContentTypePlain && contentType != enum.EmailContentTypeHTML {
		return apperror.ErrBadRequest(nil, "metadata.content_type must be text/plain or text/html", apperror.KeyEmailContentTypeInvalid)
	}
	return nil
}
