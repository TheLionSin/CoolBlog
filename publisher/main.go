package main

import (
	"context"
	"encoding/json"
	"go_blog/config"
	"go_blog/internal/events"
	"go_blog/internal/repositories"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	config.ConnectDB()

	outboxRepo := repositories.NewOutboxRepository(config.DB)

	writer := &kafka.Writer{
		Addr:         kafka.TCP("localhost:9092"),
		Topic:        "blog.events",
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}

	defer writer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("outbox publisher started")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("outbox publisher stopped")
			return
		case <-ticker.C:
			items, err := outboxRepo.FetchBatchForPublish(ctx, 50)
			if err != nil {
				log.Println("fetch outbox error:", err)
				continue
			}
			if len(items) == 0 {
				continue
			}

			for _, it := range items {
				env := events.Envelope{
					EventID:       it.EventID,
					EventType:     it.EventType,
					OccurredAt:    it.OccurredAt,
					AggregateType: it.AggregateType,
					AggregateID:   it.AggregateID,
					ActorUserID:   it.ActorUserID,
					Version:       1,
					Payload:       json.RawMessage(it.Payload),
				}

				value, err := json.Marshal(env)
				if err != nil {
					_ = outboxRepo.MarkFailed(ctx, it.ID, "marshal envelope: "+err.Error())
					continue
				}

				err = writer.WriteMessages(ctx, kafka.Message{
					Key:   []byte(it.AggregateID),
					Value: value,
					Time:  time.Now(),
				})
				if err != nil {
					_ = outboxRepo.MarkFailed(ctx, it.ID, "kafka publish: "+err.Error())
					continue
				}

				if err := outboxRepo.MarkSent(ctx, it.ID); err != nil {
					log.Println("mark sent error:", err)
				}
			}
		}

	}
}
