package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	apperror "notif-gateway/internal/value_object/handling/error"
)

type Provider interface {
	Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error)
}

type DispatcherConfig struct {
	DefaultSMS      enum.Provider
	DefaultWhatsApp enum.Provider
	DefaultEmail    enum.Provider
}

type Dispatcher struct {
	config    DispatcherConfig
	providers map[ProviderKey]Provider
	log       *slog.Logger
}

type ProviderKey struct {
	Channel  enum.Channel
	Provider enum.Provider
}

func NewDispatcher(config DispatcherConfig, providers map[ProviderKey]Provider, log *slog.Logger) *Dispatcher {
	return &Dispatcher{
		config:    config,
		providers: providers,
		log:       log,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, cmd command.Dispatch) (command.ProviderSendResult, error) {
	providerName := d.defaultProvider(cmd.Channel)
	if !providerName.Supports(cmd.Channel) {
		return command.ProviderSendResult{}, apperror.ErrBadRequest(nil, fmt.Sprintf("provider does not support channel: %s for %s", providerName, cmd.Channel), apperror.KeyProviderChannelNotSupported)
	}
	provider, ok := d.providers[ProviderKey{Channel: cmd.Channel, Provider: providerName}]
	if !ok {
		return command.ProviderSendResult{}, apperror.ErrBadRequest(nil, fmt.Sprintf("provider not found: %s for %s", providerName, cmd.Channel), apperror.KeyProviderNotFound)
	}

	result, err := provider.Send(ctx, command.ProviderSend{
		NotificationID:       cmd.NotificationID,
		Channel:              cmd.Channel,
		Provider:             providerName,
		Recipient:            cmd.Recipient,
		Body:                 cmd.Body,
		ProviderTemplateID:   cmd.ProviderTemplateID,
		ProviderTemplateVars: cmd.ProviderTemplateVars,
		Metadata:             cmd.Metadata,
	})
	if err != nil {
		d.log.WarnContext(ctx, "provider dispatch failed", slog.String("provider", providerName.String()), slog.String("channel", cmd.Channel.String()), slog.String("error", err.Error()))
		return command.ProviderSendResult{}, ensureProviderError(err)
	}
	return result, nil
}

func (d *Dispatcher) DefaultProvider(channel enum.Channel) enum.Provider {
	return d.defaultProvider(channel)
}

func (d *Dispatcher) SupportsDeliveryWebhook(channel enum.Channel, provider enum.Provider) bool {
	return provider.SupportsDeliveryWebhook(channel)
}

func (d *Dispatcher) StatusAfterSuccessfulSend(channel enum.Channel, provider enum.Provider) enum.NotificationStatus {
	return provider.StatusAfterSuccessfulSend(channel)
}

// ensureProviderError keeps *apperror.Error from providers; wraps unknown errors so the message is not lost as INTERNAL_ERROR.
func ensureProviderError(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return err
	}
	return apperror.ErrBadRequestCode(err, err.Error(), apperror.CodeProviderFailed, apperror.KeyProviderSendFailed)
}

func (d *Dispatcher) defaultProvider(channel enum.Channel) enum.Provider {
	switch channel {
	case enum.ChannelWhatsApp:
		return d.config.DefaultWhatsApp
	case enum.ChannelEmail:
		return d.config.DefaultEmail
	default:
		return d.config.DefaultSMS
	}
}
