package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/core/event"
)

type Rabbit struct {
	conn           *amqp.Connection
	log            *slog.Logger
	publishTimeout time.Duration
}

var _ event.Publisher = (*Rabbit)(nil)
var _ event.Consumer = (*RabbitConsumer)(nil)

func NewRabbit(config RabbitConfig, log *slog.Logger) (*Rabbit, error) {
	if config.DSN == "" {
		return nil, errors.New("rabbit dsn is required")
	}
	conn, err := amqp.Dial(config.DSN)
	if err != nil {
		return nil, err
	}
	rabbit := &Rabbit{conn: conn, log: log, publishTimeout: config.PublishTimeout}
	if err := rabbit.DeclareTopology(context.Background(), config); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return rabbit, nil
}

func (r *Rabbit) Close() error {
	return r.conn.Close()
}

func (r *Rabbit) Channel() (*amqp.Channel, error) {
	return r.conn.Channel()
}

func (r *Rabbit) DeclareTopology(ctx context.Context, config RabbitConfig) error {
	channel, err := r.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := channel.ExchangeDeclare(event.ExchangeDLX, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return err
	}
	for _, notifChannel := range []enum.Channel{enum.ChannelSMS, enum.ChannelWhatsApp, enum.ChannelEmail} {
		if err := declareChannelTopology(channel, notifChannel, config.RetryTTL); err != nil {
			return err
		}
	}
	return nil
}

func declareChannelTopology(channel *amqp.Channel, notifChannel enum.Channel, retryTTL time.Duration) error {
	exchange := event.ExchangeForChannel(notifChannel)
	sendQueue := event.QueueForChannel(notifChannel)
	retryQueue := event.RetryQueueForChannel(notifChannel)
	dlq := event.DeadLetterQueueForChannel(notifChannel)

	if err := channel.ExchangeDeclare(exchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(sendQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.QueueBind(sendQueue, event.RoutingKeySend, exchange, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(retryQueue, true, false, false, false, amqp.Table{
		"x-message-ttl":             int32(retryTTL.Milliseconds()),
		"x-dead-letter-exchange":    exchange,
		"x-dead-letter-routing-key": event.RoutingKeySend,
	}); err != nil {
		return err
	}
	if err := channel.QueueBind(retryQueue, event.RoutingKeyRetry, exchange, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return err
	}
	return channel.QueueBind(dlq, dlq, event.ExchangeDLX, false, nil)
}

func (r *Rabbit) Publish(ctx context.Context, message event.OutboxMessage) error {
	channel, err := r.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := channel.Confirm(false); err != nil {
		return err
	}
	confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	publishCtx, cancel := context.WithTimeout(ctx, r.publishTimeout)
	defer cancel()

	err = channel.PublishWithContext(
		publishCtx,
		event.ExchangeForChannel(message.Channel),
		event.RoutingKeySend,
		false,
		false,
		amqp.Publishing{
			ContentType:  enum.HeaderApplicationJSON,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Type:         message.Type.String(),
			Body:         body,
		},
	)
	if err != nil {
		return err
	}
	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return errors.New("rabbit publish was not acknowledged")
		}
		return nil
	case <-publishCtx.Done():
		return publishCtx.Err()
	}
}

type RabbitConsumer struct {
	rabbit *Rabbit
	log    *slog.Logger
}

func NewRabbitConsumer(rabbit *Rabbit, log *slog.Logger) *RabbitConsumer {
	return &RabbitConsumer{rabbit: rabbit, log: log}
}

func (c *RabbitConsumer) Consume(ctx context.Context, config event.ConsumerConfig, handler event.DeliveryHandler) error {
	channel, err := c.rabbit.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := channel.Qos(config.Prefetch, 0, false); err != nil {
		return err
	}

	deliveries, err := channel.Consume(event.QueueForChannel(config.Channel), "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	channelMu := &sync.Mutex{}
	workers := config.Concurrency
	if workers < 1 {
		workers = 1
	}
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case delivery, ok := <-deliveries:
					if !ok {
						errCh <- nil
						return
					}
					c.handleDelivery(ctx, channel, channelMu, delivery, config, handler)
				}
			}
		}()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (c *RabbitConsumer) handleDelivery(ctx context.Context, channel *amqp.Channel, channelMu *sync.Mutex, delivery amqp.Delivery, config event.ConsumerConfig, handler event.DeliveryHandler) {
	var message event.OutboxMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		c.log.ErrorContext(ctx, "failed to decode rabbit message", slog.String("error", err.Error()))
		channelMu.Lock()
		_ = delivery.Ack(false)
		channelMu.Unlock()
		return
	}
	if err := handler(ctx, message); err != nil {
		c.log.WarnContext(ctx, "delivery handler failed",
			slog.String("message_id", message.ID.String()),
			slog.String("notification_id", message.Payload.NotificationID.String()),
			slog.String("channel", message.Channel.String()),
			slog.Int("retry_count", retryCountFromHeaders(delivery.Headers)),
			slog.String("error", err.Error()),
		)
		c.retryOrDeadLetter(ctx, channel, channelMu, delivery, config, message)
		return
	}
	channelMu.Lock()
	_ = delivery.Ack(false)
	channelMu.Unlock()
}

func (c *RabbitConsumer) retryOrDeadLetter(ctx context.Context, channel *amqp.Channel, channelMu *sync.Mutex, delivery amqp.Delivery, config event.ConsumerConfig, message event.OutboxMessage) {
	retryCount := retryCountFromHeaders(delivery.Headers)
	headers := delivery.Headers
	if headers == nil {
		headers = amqp.Table{}
	}
	headers[enum.HeaderRetryCount] = retryCount + 1

	routingKey := event.RoutingKeyRetry
	exchange := event.ExchangeForChannel(config.Channel)
	if retryCount >= config.MaxRetry {
		exchange = event.ExchangeDLX
		routingKey = event.DeadLetterQueueForChannel(config.Channel)
	}
	channelMu.Lock()
	defer channelMu.Unlock()
	err := channel.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  enum.HeaderApplicationJSON,
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Type:         message.Type.String(),
		Headers:      headers,
		Body:         delivery.Body,
	})
	if err != nil {
		c.log.ErrorContext(ctx, "failed to publish retry or dlq message", slog.String("error", err.Error()))
		_ = delivery.Nack(false, true)
		return
	}
	_ = delivery.Ack(false)
}

func retryCountFromHeaders(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	value, ok := headers[enum.HeaderRetryCount]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	default:
		return 0
	}
}

func (r *Rabbit) String() string {
	return fmt.Sprintf("rabbit(%p)", r)
}
