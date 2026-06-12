package event

import (
	"context"

	"notif-gateway/internal/core/enum"
)

// DeliveryHandler processes one decoded queue message (e.g. DeliveryProcessor.Process).
type DeliveryHandler func(ctx context.Context, message OutboxMessage) error

// ConsumerConfig controls how the worker consumes from the broker for one logical channel.
type ConsumerConfig struct {
	Channel     enum.Channel
	Prefetch    int
	Concurrency int
	MaxRetry    int
}

// Consumer is the inbound port for subscribing to the message broker (e.g. infrastructure.RabbitConsumer).
type Consumer interface {
	Consume(ctx context.Context, config ConsumerConfig, handler DeliveryHandler) error
}
