package repositories

import (
	"context"
	"go_blog/internal/models"

	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db: db,
	}
}

func (r *RefreshTokenRepository) CreateTx(ctx context.Context, tx *gorm.DB, token *models.RefreshToken) error {
	return tx.WithContext(ctx).Create(token).Error
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("token_hash = ?", hash).
		First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

// Revoke - удаляем токен (Logout или Refresh)
func (r *RefreshTokenRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.RefreshToken{}, id).Error
}

// DeleteAllForUser - полезно для "Выйти со всех устройств" (смена пароля)
func (r *RefreshTokenRepository) DeleteAllForUser(ctx context.Context, tx *gorm.DB, userID uint) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}
