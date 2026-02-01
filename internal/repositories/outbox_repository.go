package repositories

import (
	"context"
	"go_blog/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxAttempts    = 3
	maxBackoffSecs = 120

	maxAttemptsDLQ = 1000000 // фактически бесконечно
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// Важно: вызывается ИЗ транзакции (tx)
func (r *OutboxRepository) CreateTx(ctx context.Context, tx *gorm.DB, e *models.OutboxEvent) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *OutboxRepository) CreateDLQFromDeadTx(ctx context.Context, tx *gorm.DB, dead models.OutboxEvent, lastErr string) error {
	now := time.Now().UTC()

	// В DLQ мы шлём тот же payload/envelope, но можно дополнить payload/headers уже на стороне writer
	dlq := models.OutboxEvent{
		EventID:       uuid.NewString(), // новый event_id для dlq-записи (важно!)
		Topic:         "blog.events.dlq",
		EventType:     dead.EventType,
		AggregateType: dead.AggregateType,
		AggregateID:   dead.AggregateID,
		ActorUserID:   dead.ActorUserID,
		Payload:       dead.Payload, // тот же envelope/value
		OccurredAt:    now,
		Status:        models.OutboxNew,
		Attempts:      0,
		NextAttemptAt: nil,
		LastError:     "dlq from dead: " + lastErr,
	}

	return tx.WithContext(ctx).Create(&dlq).Error
}

// Берём пачку NEW событий и "лочим" их, чтобы два publisher'а не взяли одно и то же
func (r *OutboxRepository) FetchBatchForPublish(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	var items []models.OutboxEvent
	now := time.Now().UTC()

	err := r.db.WithContext(ctx).
		Where("status = ?", models.OutboxNew).
		Where("next_attempt_at IS NULL OR next_attempt_at <= ?", now).
		Order("id asc").
		Limit(limit).
		Clauses(
			// SELECT ... FOR UPDATE SKIP LOCKED
			// gorm: это способ избежать гонок при нескольких publisher
			clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"},
		).
		Find(&items).Error

	return items, err
}

func (r *OutboxRepository) FetchDead(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	var items []models.OutboxEvent
	err := r.db.WithContext(ctx).
		Where("status = ?", models.OutboxDead).
		Order("id asc").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func (r *OutboxRepository) ResetDeadToNew(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&models.OutboxEvent{}).
		Where("id = ? AND status = ?", id, models.OutboxDead).
		Updates(map[string]any{
			"status":          models.OutboxNew,
			"attempts":        0,
			"next_attempt_at": nil,
			"last_error":      "",
		}).Error
}

func (r *OutboxRepository) MarkSent(ctx context.Context, id uint) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&models.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":          models.OutboxSent,
			"sent_at":         &now,
			"last_error":      "",
			"next_attempt_at": nil,
		}).Error
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id uint, errText string) (models.OutboxStatus, error) {
	returnStatus := models.OutboxNew

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var e models.OutboxEvent
		if err := tx.Select(
			"id", "topic", "attempts",
			"event_type", "aggregate_type", "aggregate_id",
			"actor_user_id", "payload",
		).Where("id = ?", id).First(&e).Error; err != nil {
			return err
		}

		newAttempts := e.Attempts + 1

		// DEAD

		limit := maxAttempts
		if e.Topic == "blog.events.dlq" {
			limit = maxAttemptsDLQ
		}

		if newAttempts >= limit {
			if err := tx.Model(&models.OutboxEvent{}).
				Where("id = ?", id).
				Updates(map[string]any{
					"status":          models.OutboxDead,
					"attempts":        newAttempts,
					"last_error":      errText,
					"next_attempt_at": nil,
				}).Error; err != nil {
				return err
			}

			// Создаём DLQ outbox запись ТОЛЬКО для основного топика
			// (чтобы не было dlq-of-dlq)
			if e.Topic != "blog.events.dlq" {
				if err := r.CreateDLQFromDeadTx(ctx, tx, e, errText); err != nil {
					return err
				}
			}

			returnStatus = models.OutboxDead
			return nil
		}

		// обычный backoff
		next := time.Now().UTC().Add(computeBackoff(newAttempts))
		return tx.Model(&models.OutboxEvent{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"attempts":        newAttempts,
				"last_error":      errText,
				"next_attempt_at": &next,
			}).Error
	})

	return returnStatus, err
}

func computeBackoff(attempts int) time.Duration {
	// attempts: 0..∞ (после инкремента будем использовать новое значение)
	// 1 -> 1s, 2 -> 2s, 3 -> 4s, 4 -> 8s ...
	secs := 1 << (attempts - 1)
	if secs > maxBackoffSecs {
		secs = maxBackoffSecs
	}
	return time.Duration(secs) * time.Second
}
