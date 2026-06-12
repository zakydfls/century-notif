package service

import (
	"bytes"
	"context"
	"errors"
	"text/template"

	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	apperror "notif-gateway/internal/value_object/handling/error"
)

type TemplateRenderer struct {
	tpl repository.Template
}

func NewTemplateRenderer(tpl repository.Template) *TemplateRenderer {
	return &TemplateRenderer{tpl: tpl}
}

func (r *TemplateRenderer) Render(ctx context.Context, cmd command.RenderTemplate) (command.RenderTemplateResult, error) {
	if cmd.Key == "" {
		return command.RenderTemplateResult{}, nil
	}
	tpl := &entity.NotificationTemplate{
		Key:      cmd.Key,
		Channel:  cmd.Channel,
		Provider: cmd.Provider,
	}
	if err := r.tpl.Find(ctx, tpl); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return command.RenderTemplateResult{}, apperror.ErrNotFound(err, err.Error(), apperror.KeyRecordNotFound)
		}
		return command.RenderTemplateResult{}, err
	}
	parsed, err := template.New(tpl.Key).Option("missingkey=zero").Parse(tpl.Body)
	if err != nil {
		return command.RenderTemplateResult{}, err
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, map[string]any(cmd.Params)); err != nil {
		return command.RenderTemplateResult{}, err
	}
	return command.RenderTemplateResult{
		Body:               rendered.String(),
		ProviderTemplateID: tpl.ProviderTemplateID,
	}, nil
}
