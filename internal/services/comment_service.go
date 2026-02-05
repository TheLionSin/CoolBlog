package services

import (
	"context"
	"errors"
	"go_blog/internal/dto"
	"go_blog/internal/events"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"gorm.io/gorm"
)

var ErrForbidden = errors.New("access denied")

type CommentService struct {
	db          *gorm.DB
	commentRepo *repositories.CommentRepository
	postRepo    *repositories.PostRepository // PostRepo, чтобы найти ID поста
	outboxRepo  *repositories.OutboxRepository
}

func NewCommentService(
	db *gorm.DB,
	commentRepo *repositories.CommentRepository,
	postRepo *repositories.PostRepository,
	outboxRepo *repositories.OutboxRepository) *CommentService {
	return &CommentService{
		db:          db,
		commentRepo: commentRepo,
		postRepo:    postRepo,
		outboxRepo:  outboxRepo,
	}
}

func (s *CommentService) Create(ctx context.Context, postSlug string, userID uint, text string) (*models.Comment, error) {
	// 1. Сначала ищем пост, чтобы узнать его ID и ID автора (для уведомления)
	post, err := s.postRepo.GetBySlug(ctx, postSlug)
	if err != nil {
		return nil, err
	}

	comment := &models.Comment{
		PostID: post.ID,
		UserID: userID,
		Text:   text,
	}

	// 2. Транзакция: Сохраняем коммент + Событие
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.commentRepo.CreateTx(ctx, tx, comment); err != nil {
			return err
		}

		// Событие: передаем ID автора поста, чтобы consumer мог отправить ему email/push
		evt, err := events.NewCommentCreatedEvent(comment, post.UserID, post.Slug)
		if err != nil {
			return err
		}

		return s.outboxRepo.CreateTx(ctx, tx, evt)
	})
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *CommentService) Delete(ctx context.Context, commentID uint, userID uint) error {
	// 1. Ищем коммент
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	// 2. Проверяем права (только автор может удалить)
	if comment.UserID != userID {
		return ErrForbidden
	}

	// 3. Удаляем (можно добавить Outbox событие CommentDeleted)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.commentRepo.DeleteTx(ctx, tx, comment)
	})
}

func (s *CommentService) List(ctx context.Context, postSlug string) ([]dto.CommentResponse, error) {
	post, err := s.postRepo.GetBySlug(ctx, postSlug)
	if err != nil {
		return nil, err
	}

	comments, err := s.commentRepo.ListByPostID(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	resp := make([]dto.CommentResponse, len(comments))
	for i, c := range comments {
		resp[i] = dto.CommentResponse{
			ID:        c.ID,
			Text:      c.Text,
			PostID:    c.PostID,
			UserID:    c.UserID,
			CreatedAt: c.CreatedAt,
		}
	}

	return resp, nil
}
