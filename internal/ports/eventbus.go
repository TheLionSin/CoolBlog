package ports

import (
	"context"
	"go_blog/internal/events"
)

type EventBus interface {
	Publish(ctx context.Context, e events.Envelope) error
}
