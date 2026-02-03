package services

import (
	"context"
	"errors"
	"fmt"
	"go_blog/internal/events"
	models2 "go_blog/internal/models"
	"go_blog/internal/repositories"
	"strings"

	"gorm.io/gorm"
)

type PostRepo interface {
	Create(ctx context.Context, uid uint, title, text string) (*models2.Post, error)
	UpdateOwnedBy(ctx context.Context, slug string, uid uint, updates map[string]any) (*models2.Post, error)
	DeleteOwnedBy(ctx context.Context, slug string, uid uint) error
	GetBySlug(ctx context.Context, slug string) (*models2.Post, error)
	List(ctx context.Context, page, limit int, q string) ([]models2.Post, int64, error)
}

type PostService struct {
	db     *gorm.DB
	repo   *repositories.PostRepository
	outbox *repositories.OutboxRepository
}

func NewPostService(db *gorm.DB, repo *repositories.PostRepository, outbox *repositories.OutboxRepository) *PostService {
	return &PostService{db: db, repo: repo, outbox: outbox}
}

func (s *PostService) Create(ctx context.Context, uid uint, title, text string) (*models2.Post, error) {
	title = strings.TrimSpace(title)
	text = strings.TrimSpace(text)

	var created *models2.Post

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Создаем пост в БД
		post, err := s.repo.CreateTx(ctx, tx, uid, title, text)
		if err != nil {
			return err
		}

		created = post

		// 2. Создаем событие (вся грязная работа спрятана в конструкторе)
		outboxEvent, err := events.NewPostCreatedEvent(post)
		if err != nil {
			return err
		}

		// 3. Пишем событие в БД (в той же транзакции!)
		return s.outbox.CreateTx(ctx, tx, outboxEvent)
	})

	if err != nil {
		return nil, err
	}

	return created, nil

}

func (s *PostService) Update(ctx context.Context, slug string, uid uint, title, text *string) (*models2.Post, error) {
	updates := map[string]any{}

	if title != nil {
		updates["title"] = strings.TrimSpace(*title)
	}
	if text != nil {
		updates["text"] = strings.TrimSpace(*text)
	}

	if len(updates) == 0 {
		return nil, ErrNoFieldsToUpdate
	}

	var updatedPost *models2.Post

	// START TRANSACTION
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//1. Обновляем БД
		post, err := s.repo.UpdateTx(ctx, tx, slug, uid, updates)
		if err != nil {
			return err // тут GORM вернет RecordNotFound
		}
		updatedPost = post

		//2. Создаем событие
		evt, err := events.NewPostUpdatedEvent(post)
		if err != nil {
			return err
		}

		//3. Пишем в Outbox
		return s.outbox.CreateTx(ctx, tx, evt)
	})

	if err != nil {
		return nil, err
	}

	return updatedPost, nil

}

func (s *PostService) Delete(ctx context.Context, slug string, uid uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1.Удаляем из БД
		post, err := s.repo.DeleteTx(ctx, tx, slug, uid)
		if err != nil {
			return err
		}
		//2. Создаем событие
		evt, err := events.NewPostDeletedEvent(post.ID, uid)
		if err != nil {
			return err
		}

		//3. Пишем в Outbox
		return s.outbox.CreateTx(ctx, tx, evt)
	})
}

func (s *PostService) Get(ctx context.Context, slug string) (*models2.Post, error) {
	post, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return post, nil
}

func (s *PostService) List(ctx context.Context, page, limit int, q string) ([]models2.Post, int64, error) {
	return s.repo.List(ctx, page, limit, q)
}

func uintToString(v uint) string {
	// не идеально, но ок для старта. Потом приведём к нормальному виду под твои модели/ID
	return fmt.Sprint(v)
}
