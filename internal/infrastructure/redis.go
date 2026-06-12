package infrastructure

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis interface {
	GetClient() *redis.Client
	Close() error
}

func NewRedis(config RedisConfig) (Redis, error) {
	if config.DSN == "" {
		return nil, errors.New("redis dsn is required")
	}
	opts, err := redis.ParseURL(config.DSN)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err = client.Ping(ctx).Result(); err != nil {
		return nil, err
	}
	return &redisW{
		client: client,
	}, nil
}

type redisW struct {
	client *redis.Client
}

func (infra *redisW) GetClient() *redis.Client {
	return infra.client
}

func (infra *redisW) Close() error {
	slog.Info("Closing redis connection...")
	return infra.client.Close()
}
