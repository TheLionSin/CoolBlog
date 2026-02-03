package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go_blog/config"
	"go_blog/internal/events"
	"go_blog/internal/metrics"
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

	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	eventsWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokers),
		Topic:        TopicEvents,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer eventsWriter.Close()

	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokers),
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
		case <-ctx.Done(): // ctx - это signal context
			log.Println("shutdown requested, stopping fetch loop")
			return

		case <-ticker.C:
			// 1. Fetch делаем с таймаутом, но независимым от shutdown (или проверяем ctx перед этим)
			// Если ctx отменен, мы просто не пойдем в базу.
			if ctx.Err() != nil {
				return
			}

			fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			items, err := outboxRepo.FetchBatchForPublish(fetchCtx, 50)
			cancel()

			if err != nil {
				log.Println("fetch outbox error:", err)
				continue
			}
			if len(items) == 0 {
				continue
			}

			metrics.OutboxFetched.Add(1)

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

					metrics.OutboxRetry.Add(1)

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
				publishCtx, cancelPub := context.WithTimeout(context.Background(), 10*time.Second)
				err = w.WriteMessages(publishCtx, msg)
				cancelPub()

				if err != nil {
					log.Printf("OUTBOX FAIL id=%d event_id=%s stage=kafka_publish topic=%s err=%v", it.ID, it.EventID, it.Topic, err)

					metrics.OutboxRetry.Add(1)

					failCtx, cancelFail := context.WithTimeout(context.Background(), 5*time.Second)
					_, _ = outboxRepo.MarkFailed(failCtx, it.ID, "kafka publish: "+err.Error())
					cancelFail()
					continue
				}

				// 6) Mark sent
				dbCtx, cancelDb := context.WithTimeout(context.Background(), 5*time.Second)
				if err := outboxRepo.MarkSent(dbCtx, it.ID); err != nil {
					log.Printf("CRITICAL: Message sent to Kafka but failed to update DB for id=%d: %v", it.ID, err)
				}
				cancelDb()

				metrics.OutboxSent.Add(1)

				lag := time.Since(it.OccurredAt)

				log.Printf(
					"OUTBOX SENT id=%d event_id=%s topic=%s lag=%s attempts=%d",
					it.ID, it.EventID, it.Topic, lag, it.Attempts,
				)
			}
		}
	}
}

func hdr(key, val string) kafka.Header {
	return kafka.Header{Key: key, Value: []byte(val)}
}
