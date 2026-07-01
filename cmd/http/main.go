package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"notif-gateway/internal/adapter/inbound/rest"
	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/adapter/provider"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/service"
	"notif-gateway/internal/infrastructure"
	applogger "notif-gateway/internal/pkg/logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := infrastructure.NewConfig()
	if err != nil {
		panic(err)
	}
	log := applogger.New(applogger.Config{Level: config.Logger.Level, Format: config.Logger.Format})
	slog.SetDefault(log)

	postgres, err := infrastructure.NewPostgres(config.Postgres)
	if err != nil {
		log.Error("failed to connect postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer postgres.Close()
	if config.Migration.AutoMigrate && (config.App.Env == enum.EnvLocal || config.App.Env == enum.EnvDevelopment) {
		if err := postgres.AutoMigrate(); err != nil {
			log.Error("failed to auto migrate", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	db := postgres.DB()
	notifRepo := repository.NewNotification(db)
	outboxRepo := repository.NewNotificationOutbox(db)
	templateRepo := repository.NewTemplate(db)
	webhookEventRepo := repository.NewWebhookEvent(db)
	txm := repository.NewTxManager(db)

	providers := provider.NewRegistry(config.Provider)
	dispatcher := service.NewDispatcher(service.DispatcherConfig{
		DefaultSMS:      config.Provider.DefaultSMS,
		DefaultWhatsApp: config.Provider.DefaultWhatsApp,
		DefaultEmail:    config.Provider.DefaultEmail,
	}, providers, log)

	renderer := service.NewTemplateRenderer(templateRepo)
	builder := service.NewNotificationBuilder(renderer, dispatcher)
	notifications := service.NewNotificationService(
		notifRepo,
		outboxRepo,
		txm,
		builder,
	)
	webhooks := service.NewWebhookHandler(notifRepo, webhookEventRepo, txm)

	twilioWebhook := rest.TwilioWebhook{
		AuthToken:         strings.TrimSpace(config.Provider.Twilio.AuthToken),
		BaseURL:           strings.TrimSpace(config.Provider.Twilio.WebhookBaseURL),
		ValidateSignature: config.Provider.Twilio.ValidateWebhookSignature,
	}
	if twilioWebhook.ValidateSignature && twilioWebhook.AuthToken == "" {
		log.Warn("TWILIO_VALIDATE_WEBHOOK_SIGNATURE enabled but TWILIO_AUTH_TOKEN is empty; Twilio webhook signatures will not be validated")
	}
	wahaWebhook := rest.WAHAWebhook{
		HMACKey:           strings.TrimSpace(config.Provider.WAHA.WebhookHMACKey),
		ValidateSignature: config.Provider.WAHA.ValidateWebhookSignature,
	}
	if wahaWebhook.ValidateSignature && wahaWebhook.HMACKey == "" {
		log.Warn("WAHA_VALIDATE_WEBHOOK_SIGNATURE enabled but WAHA_WEBHOOK_HMAC_KEY is empty; WAHA webhooks will be rejected")
	}
	resendWebhook := rest.ResendWebhook{
		Secret:            strings.TrimSpace(config.Provider.Resend.WebhookSecret),
		ValidateSignature: config.Provider.Resend.ValidateWebhookSignature,
	}
	if resendWebhook.ValidateSignature && resendWebhook.Secret == "" {
		log.Warn("RESEND_VALIDATE_WEBHOOK_SIGNATURE enabled but RESEND_WEBHOOK_SECRET is empty; Resend webhooks will be rejected")
	}
	handler := rest.NewHandler(notifications, webhooks, log, config.HTTP.NotificationAPIKey, twilioWebhook, wahaWebhook, resendWebhook)
	server := &http.Server{
		Addr:    config.HTTP.Addr,
		Handler: handler.Router(),
	}

	go func() {
		log.Info("http server starting", slog.String("addr", config.HTTP.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", slog.String("error", err.Error()))
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.HTTP.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", slog.String("error", err.Error()))
	}
}
