package provider

import (
	emailgodaddy "notif-gateway/internal/adapter/outbound/email/godaddy"
	emailresend "notif-gateway/internal/adapter/outbound/email/resend"
	smsalibaba "notif-gateway/internal/adapter/outbound/sms/alibaba"
	smstwilio "notif-gateway/internal/adapter/outbound/sms/twilio"
	watwilio "notif-gateway/internal/adapter/outbound/whatsapp/twilio"
	"notif-gateway/internal/adapter/outbound/whatsapp/waha"
	"notif-gateway/internal/adapter/outbound/whatsapp/whatsockt"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/service"
	"notif-gateway/internal/infrastructure"
)

func NewRegistry(config infrastructure.ProviderConfig) map[service.ProviderKey]service.Provider {
	return map[service.ProviderKey]service.Provider{
		// SMS
		{Channel: enum.ChannelSMS, Provider: enum.ProviderTwilio}:  smstwilio.NewProvider(config.Twilio),
		{Channel: enum.ChannelSMS, Provider: enum.ProviderAlibaba}: smsalibaba.NewProvider(config.Alibaba),

		// WhatsApp
		{Channel: enum.ChannelWhatsApp, Provider: enum.ProviderTwilio}:    watwilio.NewProvider(config.Twilio),
		{Channel: enum.ChannelWhatsApp, Provider: enum.ProviderWAHA}:      waha.NewProvider(config.WAHA),
		{Channel: enum.ChannelWhatsApp, Provider: enum.ProviderWhatsockt}: whatsockt.NewProvider(config.Whatsockt),

		// Email
		{Channel: enum.ChannelEmail, Provider: enum.ProviderGoDaddy}: emailgodaddy.NewProvider(config.GoDaddy),
		{Channel: enum.ChannelEmail, Provider: enum.ProviderResend}:  emailresend.NewProvider(config.Resend),
	}
}
