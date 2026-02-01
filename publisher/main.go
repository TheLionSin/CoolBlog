package main

import (
	"context"
	"encoding/json"
	"fmt"
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

const (
	TopicEvents = "blog.events"
	TopicDLQ    = "blog.events.dlq"
)

func main() {
	config.ConnectDB()

	outboxRepo := repositories.NewOutboxRepository(config.DB)

	eventsWriter := &kafka.Writer{
		Addr:         kafka.TCP("localhost:9092"),
		Topic:        TopicEvents,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer eventsWriter.Close()

	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP("localhost:9092"),
		Topic:        TopicDLQ,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer dlqWriter.Close()

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
				// 1) Собираем envelope
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

				// 2) Marshal
				value, err := json.Marshal(env)
				if err != nil {
					log.Printf("OUTBOX FAIL id=%d event_id=%s stage=marshal err=%v", it.ID, it.EventID, err)
					_, _ = outboxRepo.MarkFailed(ctx, it.ID, "marshal envelope: "+err.Error())
					continue
				}

				// 3) Выбираем writer по it.Topic
				w := eventsWriter
				if it.Topic == TopicDLQ {
					w = dlqWriter
				}

				// 4) Формируем kafka message
				msg := kafka.Message{
					Key:   []byte(it.AggregateID),
					Value: value,
					Time:  time.Now(),
				}

				// DLQ headers: удобно расследовать прямо в Kafka
				if it.Topic == TopicDLQ {
					msg.Headers = []kafka.Header{
						hdr("source_topic", TopicEvents),
						hdr("outbox_id", fmt.Sprintf("%d", it.ID)),
						hdr("event_id", it.EventID),
						hdr("attempts", fmt.Sprintf("%d", it.Attempts)),
						hdr("last_error", it.LastError),
						hdr("published_at", time.Now().UTC().Format(time.RFC3339Nano)),
					}
				}

				// 5) Publish
				err = w.WriteMessages(ctx, msg)
				if err != nil {
					log.Printf("OUTBOX FAIL id=%d event_id=%s stage=kafka_publish topic=%s err=%v", it.ID, it.EventID, it.Topic, err)
					_, _ = outboxRepo.MarkFailed(ctx, it.ID, "kafka publish: "+err.Error())
					continue
				}

				// 6) Mark sent
				if err := outboxRepo.MarkSent(ctx, it.ID); err != nil {
					log.Println("mark sent error:", err)
					continue
				}

				log.Printf("OUTBOX SENT id=%d event_id=%s topic=%s", it.ID, it.EventID, it.Topic)
			}
		}
	}
}

func hdr(key, val string) kafka.Header {
	return kafka.Header{Key: key, Value: []byte(val)}
}
