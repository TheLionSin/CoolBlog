package main

import (
	"context"
	"encoding/json"
	"errors"
	"go_blog/config"
	"go_blog/internal/events"
	"go_blog/internal/metrics"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	db, _ := config.ConnectDB()

	auditRepo := repositories.NewAuditLogRepository(db)

	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	log.Println("KAFKA_BROKERS =", brokers)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{brokers},
		GroupTopics: []string{"blog.events", "blog.users", "blog.events.dlq", "blog.comments", "blog.likes"},
		GroupID:     "audit-log-consumer",
	})

	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("audit consumer started")

	for {
		log.Println("waiting message...")
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("consumer shutdown: context canceled")
				return
			}

			log.Printf("fetch error: %v", err)
			metrics.ConsumerErrors.Add(1)
			continue
		}

		var env events.Envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			log.Println("invalid event:", err)
			metrics.ConsumerErrors.Add(1)
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
			// Проверяем: это дубликат?
			// (Для простоты проверяем текст, по-хорошему надо проверять код ошибки Postgres 23505)
			isDuplicate := false
			// В GORM/Postgres ошибка часто содержит этот текст
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
				isDuplicate = true
			}

			if isDuplicate {
				log.Printf("Event %s already processed, skipping", env.EventID)
				// Мы НЕ делаем continue, мы идем дальше к коммиту!
			} else {
				// Реальная ошибка (нет подключения к БД и т.д.)
				// Вот тут мы не можем коммитить, надо ретраить или падать
				log.Println("failed to save audit log:", err)
				metrics.ConsumerErrors.Add(1)

				time.Sleep(time.Second)
				continue
			}
		}

		log.Printf(
			"CONSUMER processed event_id=%s partition=%d offset=%d",
			env.EventID,
			msg.Partition,
			msg.Offset,
		)

		// Коммит делаем с независимым таймаутом, чтобы он успел завершиться
		// даже если основной ctx (приложение) уже закрывается
		commitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = reader.CommitMessages(commitCtx, msg)
		cancel() // Всегда вызываем cancel, чтобы не было утечки контекста

		if err != nil {
			log.Println("failed to commit message:", err)
			metrics.ConsumerErrors.Add(1)
			continue
		}

		// Только после успешного коммита считаем сообщение полностью обработанным
		metrics.ConsumerProcessed.Add(1)

		log.Printf("CONSUMER processed event_id=%s offset=%d", env.EventID, msg.Offset)

	}
}
