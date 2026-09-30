package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"notif-gateway/internal/adapter/outbound/repository"
	"notif-gateway/internal/adapter/provider"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/event"
	"notif-gateway/internal/core/service"
	"notif-gateway/internal/infrastructure"
	applogger "notif-gateway/internal/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("application stopped gracefully")
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := infrastructure.NewConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := applogger.New(applogger.Config{Level: config.Logger.Level, Format: config.Logger.Format})
	slog.SetDefault(logger)

	postgres, err := infrastructure.NewPostgres(config.Postgres)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer func() {
		logger.Info("closing postgres connection")
		postgres.Close()
	}()

	rabbit, err := infrastructure.NewRabbit(config.Rabbit, logger)
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer func() {
		logger.Info("closing rabbitmq connection")
		rabbit.Close()
	}()

	providers := provider.NewRegistry(config.Provider)
	dispatcher := service.NewDispatcher(service.DispatcherConfig{
		DefaultSMS:      config.Provider.DefaultSMS,
		DefaultWhatsApp: config.Provider.DefaultWhatsApp,
		DefaultEmail:    config.Provider.DefaultEmail,
	}, providers, logger)

	db := postgres.DB()
	notifRepo := repository.NewNotification(db)
	attemptRepo := repository.NewAttempt(db)
	templateRepo := repository.NewTemplate(db)
	outboxRepo := repository.NewNotificationOutbox(db)
	builder := service.NewNotificationBuilder(service.NewTemplateRenderer(templateRepo), dispatcher, service.EmailSecurityPolicy{
		BlockDisposable:     config.EmailSecurity.BlockDisposable,
		DisposableAllowlist: config.EmailSecurity.DisposableAllowlist,
	})
	processor := service.NewDeliveryProcessor(notifRepo, attemptRepo, dispatcher, builder)
	var consumer event.Consumer = infrastructure.NewRabbitConsumer(rabbit, logger)
	sweeper := service.NewOutboxSweeper(outboxRepo, rabbit, logger, service.OutboxSweeperConfig{
		SweepInterval:  config.Outbox.SweepInterval,
		BatchSize:      config.Outbox.BatchSize,
		LockTTL:        config.Outbox.LockTTL,
		RetryBaseDelay: config.Outbox.RetryBaseDelay,
		RetryMaxDelay:  config.Outbox.RetryMaxDelay,
	})

	workers := infrastructure.NewChannelWorkers(infrastructure.ChannelWorkerConfig{
		Prefetch:    config.Worker.Prefetch,
		Concurrency: config.Worker.Concurrency,
		MaxRetry:    config.Worker.MaxRetry,
	})

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		logger.Info("outbox sweeper starting",
			slog.Duration("sweep_interval", config.Outbox.SweepInterval),
			slog.Int("batch_size", config.Outbox.BatchSize),
			slog.Duration("lock_ttl", config.Outbox.LockTTL),
		)
		return sweeper.Run(ctx)
	})
	for _, ch := range []enum.Channel{enum.ChannelSMS, enum.ChannelWhatsApp, enum.ChannelEmail} {
		workerCfg := workers.ForChannel(ch)
		group.Go(func() error {
			logger.Info("mq consumer starting",
				slog.String("channel", ch.String()),
				slog.Int("prefetch", workerCfg.Prefetch),
				slog.Int("concurrency", workerCfg.Concurrency),
				slog.Int("max_retry", workerCfg.MaxRetry),
			)

			err := consumer.Consume(ctx, event.ConsumerConfig{
				Channel:     ch,
				Prefetch:    workerCfg.Prefetch,
				Concurrency: workerCfg.Concurrency,
				MaxRetry:    workerCfg.MaxRetry,
			}, processor.Process)

			if errors.Is(err, context.Canceled) {
				logger.Info("mq consumer stopped (context canceled)", slog.String("channel", ch.String()))
				return nil
			}
			if err != nil {
				return fmt.Errorf("consumer channel=%s: %w", ch.String(), err)
			}

			logger.Info("mq consumer stopped", slog.String("channel", ch.String()))
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return fmt.Errorf("consumer group: %w", err)
	}
	return nil
}
