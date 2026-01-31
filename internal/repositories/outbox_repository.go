package repositories

import (
	"context"
	"go_blog/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxAttempts    = 10
	maxBackoffSecs = 120
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

func (r *OutboxRepository) MarkFailed(ctx context.Context, id uint, errText string) error {
	// 1) Сначала считаем новое количество попыток (берём текущее Attempts из БД)
	var e models.OutboxEvent
	if err := r.db.WithContext(ctx).
		Select("id", "attempts").
		Where("id = ?", id).
		First(&e).Error; err != nil {
		return err
	}

	newAttempts := e.Attempts + 1

	if newAttempts >= maxAttempts {
		return r.db.WithContext(ctx).
			Model(&models.OutboxEvent{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"status":          models.OutboxDead,
				"attempts":        newAttempts,
				"last_error":      errText,
				"next_attempt_at": nil,
			}).Error
	}

	next := time.Now().UTC().Add(computeBackoff(newAttempts))

	return r.db.WithContext(ctx).
		Model(&models.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":        newAttempts,
			"last_error":      errText,
			"next_attempt_at": &next,
		}).Error
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
