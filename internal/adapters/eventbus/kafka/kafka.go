package kafka

import (
	"context"
	"encoding/json"
	"go_blog/internal/events"
	"time"

	"github.com/segmentio/kafka-go"
)

type EventBus struct {
	writer *kafka.Writer
	topic  string
}

func New(brokers []string, topic string) *EventBus {
	return &EventBus{
		topic: topic,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			Async:        false,
		},
	}
}

func (b *EventBus) Publish(ctx context.Context, e events.Envelope) error {
	value, err := json.Marshal(e)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(e.AggregateID),
		Value: value,
		Time:  time.Now(),
	}

	return b.writer.WriteMessages(ctx, msg)
}
