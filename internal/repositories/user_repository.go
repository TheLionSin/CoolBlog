package repositories

import (
	"context"
	"errors"
	"go_blog/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrUserExists = errors.New("user already exists")

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateTx(ctx context.Context, tx *gorm.DB, user *models.User) error {
	err := tx.WithContext(ctx).Create(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrUserExists
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserExists
		}
		return err
	}
	return nil
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrUserExists
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUserExists
		}
		return err
	}
	return nil
}

func (r *UserRepository) UpdateTx(ctx context.Context, tx *gorm.DB, user *models.User, updates map[string]any) error {
	return tx.WithContext(ctx).Model(user).Updates(updates).Error
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ? AND is_active = ?", email, true).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
