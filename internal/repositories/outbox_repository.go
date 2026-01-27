package repositories

import (
	"context"
	"go_blog/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	err := r.db.WithContext(ctx).
		Where("status = ?", models.OutboxNew).
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
			"status":     "SENT",
			"sent_at":    &now,
			"last_error": "",
		}).Error
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id uint, errText string) error {
	return r.db.WithContext(ctx).
		Model(&models.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": errText,
		}).Error
}
