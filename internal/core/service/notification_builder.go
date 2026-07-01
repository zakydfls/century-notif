package service

import (
	"context"
	"strings"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/enum"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

type NotificationBuilder struct {
	renderer   *TemplateRenderer
	dispatcher *Dispatcher
}

func NewNotificationBuilder(renderer *TemplateRenderer, dispatcher *Dispatcher) *NotificationBuilder {
	return &NotificationBuilder{
		renderer:   renderer,
		dispatcher: dispatcher,
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
