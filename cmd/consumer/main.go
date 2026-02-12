package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go_blog/config"
	"go_blog/internal/events"
	"go_blog/internal/metrics"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

	metrics.Init()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{brokers},
		GroupTopics: []string{"blog.events", "blog.users", "blog.events.dlq", "blog.comments", "blog.likes"},
		GroupID:     "audit-log-consumer",
	})

	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("audit consumer started")

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		// API: 8080
		// Publisher: 9091 (например)
		// Consumer: 9092
		log.Println("Metrics server started on :9092")
		if err := http.ListenAndServe(":9092", nil); err != nil {
			log.Println("Metrics server failed:", err)
		}
	}()

	for {
		log.Println("waiting message...")
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("consumer shutdown: context canceled")
				return
			}
			log.Printf("fetch error: %v", err)
			metrics.ConsumerErrors.Inc()
			continue
		}

		var env events.Envelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			log.Println("invalid event:", err)
			metrics.ConsumerErrors.Inc()
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
			var pgErr *pgconn.PgError
			// Проверяем: это дубликат?
			if errors.As(err, &pgErr) && pgErr.Code == "23505" { //Либо так if errors.Is(err, gorm.ErrDuplicatedKey)
				log.Printf("Duplicate event %s, skipping", env.EventID)
				// ВНИМАНИЕ: Здесь нет 'continue' и нет 'return'.
				// Мы просто выходим из if/else и идем вниз -> к reader.CommitMessages
				// Это и есть Idempotency (Идемпотентность).
			} else {
				// Реальная ошибка (нет подключения к БД и т.д.)
				// Вот тут мы не можем коммитить, надо ретраить или падать
				log.Println("failed to save audit log:", err)
				metrics.ConsumerErrors.Inc()

				// Ждем и пробуем снова (Retry Policy)
				time.Sleep(time.Second)
				continue // Возвращаемся в начало цикла, НЕ делаем коммит
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
			metrics.ConsumerErrors.Inc()
			continue
		}

		// Только после успешного коммита считаем сообщение полностью обработанным
		metrics.ConsumerProcessed.Inc()

		log.Printf("CONSUMER processed event_id=%s offset=%d", env.EventID, msg.Offset)

	}
}
