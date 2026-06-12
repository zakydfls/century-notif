package event

import "context"

// Publisher is the outbound port for enqueueing notification work to a message broker.
type Publisher interface {
	Publish(ctx context.Context, message OutboxMessage) error
}
