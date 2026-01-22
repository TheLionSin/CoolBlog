package repositories

import (
	"context"
	"errors"
	"go_blog/models"

	"gorm.io/gorm"
)

var ErrAlreadyLiked = errors.New("already liked")

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

func (r *LikeRepository) postIDBySlug(ctx context.Context, slug string) (uint, error) {
	var post models.Post
	if err := r.db.WithContext(ctx).Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		return 0, err
	}
	return post.ID, nil
}

func (r *LikeRepository) Like(ctx context.Context, postSlug string, userID uint) error {
	postID, err := r.postIDBySlug(ctx, postSlug)
	if err != nil {
		return err
	}

	like := models.PostLike{PostID: postID, UserID: userID}
	if err := r.db.WithContext(ctx).Create(&like).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAlreadyLiked
		}
		return err
	}
	return nil
}

func (r *LikeRepository) Unlike(ctx context.Context, postSlug string, userID uint) error {
	postID, err := r.postIDBySlug(ctx, postSlug)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Unscoped().Where("post_id = ? AND user_id = ?", postID, userID).Delete(&models.PostLike{}).Error
}

func (r *LikeRepository) CountByPostSlug(ctx context.Context, postSlug string) (int64, error) {
	postID, err := r.postIDBySlug(ctx, postSlug)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.PostLike{}).Where("post_id = ?", postID).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
