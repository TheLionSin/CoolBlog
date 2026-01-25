package inmem

import (
	"context"
	"go_blog/internal/events"
	"sync"
)

type Bus struct {
	mu     sync.Mutex
	Events []events.Envelope
}

func New() *Bus {
	return &Bus{}
}

func (b *Bus) Publish(ctx context.Context, e events.Envelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Events = append(b.Events, e)
	return nil
}
