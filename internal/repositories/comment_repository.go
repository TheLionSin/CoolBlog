package repositories

import (
	"context"
	"go_blog/internal/models"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) CreateTx(ctx context.Context, tx *gorm.DB, comment *models.Comment) error {
	return tx.WithContext(ctx).Create(comment).Error
}

func (r *CommentRepository) GetByID(ctx context.Context, id uint) (*models.Comment, error) {
	var comment models.Comment
	if err := r.db.WithContext(ctx).First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// DeleteTx - удаление в транзакции (на случай, если захотим событие CommentDeleted)
func (r *CommentRepository) DeleteTx(ctx context.Context, tx *gorm.DB, comment *models.Comment) error {
	return tx.WithContext(ctx).Delete(comment).Error
}

func (r *CommentRepository) ListByPostID(ctx context.Context, postID uint) ([]models.Comment, error) {
	var comments []models.Comment
	if err := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Order("created_at asc").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}
