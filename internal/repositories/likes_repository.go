package repositories

import (
	"context"
	"errors"
	"go_blog/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrAlreadyLiked = errors.New("already liked")

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

// AddTx - Добавить лайк в транзакции
func (r *LikeRepository) AddTx(ctx context.Context, tx *gorm.DB, like *models.PostLike) error {
	if err := tx.WithContext(ctx).Create(like).Error; err != nil {
		// Обработка уникальности (один юзер - один лайк на пост)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAlreadyLiked
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyLiked
		}
		return err
	}
	return nil
}

// RemoveTx - Удалить лайк (Unlike)
func (r *LikeRepository) RemoveTx(ctx context.Context, tx *gorm.DB, postID, userID uint) error {
	result := tx.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&models.PostLike{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *LikeRepository) CountByPostID(ctx context.Context, postID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PostLike{}).
		Where("post_id = ?", postID).
		Count(&count).Error
	return count, err
}

// HasUserLiked - вспомогательный метод (для фронта: подсвечивать сердечко или нет)
func (r *LikeRepository) HasUserLiked(ctx context.Context, postID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PostLike{}).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Count(&count).Error
	return count > 0, err
}
