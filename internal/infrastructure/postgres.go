package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"notif-gateway/internal/core/entity"
)

type Postgres struct {
	db *gorm.DB
}

func NewPostgres(config PostgresConfig) (*Postgres, error) {
	if config.DSN == "" {
		return nil, errors.New("postgres dsn is required")
	}
	db, err := gorm.Open(postgres.Open(config.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogMode(config.GormLogLevel)),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}
	return &Postgres{db: db}, nil
}

func (p *Postgres) DB() *gorm.DB {
	return p.db
}

func (p *Postgres) AutoMigrate() error {
	return p.db.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	}).AutoMigrate(
		&entity.Notification{},
		&entity.NotificationAttempt{},
		&entity.NotificationWebhookEvent{},
		&entity.NotificationTemplate{},
	)
}

// gormLogMode maps config string to GORM logger level (info logs SQL).
func gormLogMode(level string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	case "info", "debug":
		return logger.Info
	default:
		return logger.Warn
	}
}

func (p *Postgres) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
