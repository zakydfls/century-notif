package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
)

type providerFunc func(context.Context, command.ProviderSend) (command.ProviderSendResult, error)

func (f providerFunc) Send(ctx context.Context, send command.ProviderSend) (command.ProviderSendResult, error) {
	return f(ctx, send)
}

func TestDispatcherUsesNotificationProvider(t *testing.T) {
	t.Parallel()

	called := ""
	dispatcher := NewDispatcher(DispatcherConfig{DefaultWhatsApp: enum.ProviderWhatsockt}, map[ProviderKey]Provider{
		{Channel: enum.ChannelWhatsApp, Provider: enum.ProviderTwilio}: providerFunc(func(_ context.Context, _ command.ProviderSend) (command.ProviderSendResult, error) {
			called = enum.ProviderTwilio.String()
			return command.ProviderSendResult{}, nil
		}),
		{Channel: enum.ChannelWhatsApp, Provider: enum.ProviderWhatsockt}: providerFunc(func(_ context.Context, _ command.ProviderSend) (command.ProviderSendResult, error) {
			called = enum.ProviderWhatsockt.String()
			return command.ProviderSendResult{}, nil
		}),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := dispatcher.Dispatch(context.Background(), command.Dispatch{
		Channel: enum.ChannelWhatsApp, Provider: enum.ProviderTwilio,
	})
	if err != nil {
		t.Fatal(err)
	}
	if called != enum.ProviderTwilio.String() {
		t.Fatalf("called provider = %q, want twilio", called)
	}
	_, err = dispatcher.Dispatch(context.Background(), command.Dispatch{Channel: enum.ChannelWhatsApp})
	if err != nil {
		t.Fatal(err)
	}
	if called != enum.ProviderWhatsockt.String() {
		t.Fatalf("fallback provider = %q, want whatsockt", called)
	}
}
