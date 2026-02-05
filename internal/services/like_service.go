package services

import (
	"context"
	"go_blog/internal/events"
	"go_blog/internal/models"
	"go_blog/internal/repositories"

	"gorm.io/gorm"
)

var ErrAlreadyLiked = repositories.ErrAlreadyLiked

type LikeService struct {
	db         *gorm.DB
	likeRepo   *repositories.LikeRepository
	postRepo   *repositories.PostRepository
	outboxRepo *repositories.OutboxRepository
}

func NewLikeService(
	db *gorm.DB,
	likeRepo *repositories.LikeRepository,
	postRepo *repositories.PostRepository,
	outboxRepo *repositories.OutboxRepository,
) *LikeService {
	return &LikeService{
		db:         db,
		likeRepo:   likeRepo,
		postRepo:   postRepo,
		outboxRepo: outboxRepo,
	}
}

func (s *LikeService) Like(ctx context.Context, postSlug string, userID uint) error {
	//1. Ищем пост
	post, err := s.postRepo.GetBySlug(ctx, postSlug)
	if err != nil {
		return err
	}

	like := &models.PostLike{
		PostID: post.ID,
		UserID: userID,
	}

	// 2. Транзакция: Лайк + Событие
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.likeRepo.AddTx(ctx, tx, like); err != nil {
			return err
		}

		//Событие для уведомлений
		evt, err := events.NewPostLikedEvent(like, post.UserID, post.Slug)
		if err != nil {
			return err
		}

		return s.outboxRepo.CreateTx(ctx, tx, evt)
	})
}

func (s *LikeService) Unlike(ctx context.Context, postSlug string, userID uint) error {
	post, err := s.postRepo.GetBySlug(ctx, postSlug)
	if err != nil {
		return err
	}

	// Unlike можно делать без транзакции (если не нужно событие Unliked)
	// Но для порядка можно и в транзакции, или просто передать s.db
	return s.likeRepo.RemoveTx(ctx, s.db, post.ID, userID)
}

func (s *LikeService) GetLikesCount(ctx context.Context, postSlug string) (int64, error) {
	post, err := s.postRepo.GetBySlug(ctx, postSlug)
	if err != nil {
		return 0, err
	}
	return s.likeRepo.CountByPostID(ctx, post.ID)
}
