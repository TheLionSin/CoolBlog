package repositories

import (
	"context"
	"errors"
	"go_blog/models"
	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) postIDBySlug(ctx context.Context, slug string) (uint, error) {
	var post models.Post
	if err := r.db.WithContext(ctx).Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		return 0, err
	}
	return post.ID, nil
}

func (r *CommentRepository) Create(ctx context.Context, postSlug string, userID uint, text string) (*models.Comment, error) {
	postID, err := r.postIDBySlug(ctx, postSlug)
	if err != nil {
		return nil, err
	}

	comment := &models.Comment{
		PostID: postID,
		UserID: userID,
		Text:   text,
	}

	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

func (r *CommentRepository) DeleteOwnedBy(ctx context.Context, commentID, userID uint) error {
	var comment models.Comment
	if err := r.db.WithContext(ctx).Where("id = ?", commentID).First(&comment).Error; err != nil {
		return err
	}

	if comment.UserID != userID {
		return ErrForbidden
	}

	return r.db.WithContext(ctx).Delete(&comment).Error
}

func (r *CommentRepository) ListByPostSlug(ctx context.Context, postSlug string) ([]models.Comment, error) {
	postID, err := r.postIDBySlug(ctx, postSlug)
	if err != nil {
		return nil, err
	}

	var comments []models.Comment
	if err := r.db.WithContext(ctx).Where("post_id = ?", postID).Order("created_at asc").Find(&comments).Error; err != nil {
		return nil, err
	}

	return comments, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

var ErrForbidden = errors.New("forbidden")
