package main

import (
	"log/slog"
	"os"

	"notif-gateway/internal/infrastructure"
	applogger "notif-gateway/internal/pkg/logger"
)

func main() {
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

	log.Info("database migration starting")
	if err := postgres.AutoMigrate(); err != nil {
		log.Error("database migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("database migration completed")
}
