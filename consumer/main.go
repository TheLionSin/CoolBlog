package main

import (
	"context"
	"encoding/json"
	"go_blog/config"
	"go_blog/internal/events"
	"go_blog/internal/repositories"
	"go_blog/models"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
)

func main() {
	config.ConnectDB()

	db := config.DB

	auditRepo := repositories.NewAuditLogRepository(db)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "blog.events",
		GroupID: "audit-log-consumer",
	})

	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("audit consumer started")

	for {
		log.Println("waiting message...")
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			log.Printf("fetch error: %v", err)
			continue
		}
		log.Printf("got message offset=%d", msg.Offset)

		var env events.Envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			log.Println("invalid event:", err)
			continue
		}

		logEntry := models.AuditLog{
			EventID:       env.EventID,
			EventType:     env.EventType,
			AggregateType: env.AggregateType,
			AggregateID:   env.AggregateID,
			ActorUserID:   env.ActorUserID,
			Payload:       string(env.Payload),
			OccurredAt:    env.OccurredAt,
		}

		if err := auditRepo.Create(ctx, &logEntry); err != nil {
			log.Println("failed to save audit log:", err)
			continue
		}

		log.Printf("processing event %s offset=%d", env.EventID, msg.Offset)

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Println("failed to commit message:", err)
		}
	}
}
