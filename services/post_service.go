package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go_blog/internal/events"
	"go_blog/internal/ports"
	"go_blog/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepo interface {
	Create(ctx context.Context, uid uint, title, text string) (*models.Post, error)
	UpdateOwnedBy(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error)
	DeleteOwnedBy(ctx context.Context, slug string, uid uint) error
	GetBySlug(ctx context.Context, slug string) (*models.Post, error)
	List(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error)
}

type PostService struct {
	repo PostRepo
	bus  ports.EventBus
}

func NewPostService(repo PostRepo, bus ports.EventBus) *PostService {
	return &PostService{repo: repo, bus: bus}
}

func (s *PostService) Create(ctx context.Context, uid uint, title, text string) (*models.Post, error) {
	title = strings.TrimSpace(title)
	text = strings.TrimSpace(text)

	post, err := s.repo.Create(ctx, uid, title, text)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(events.PostCreatedPayload{
		PostID: uintToString(post.ID),
		Title:  post.Title,
		Slug:   post.Slug,
	})
	if err != nil {
		return nil, err
	}

	if err := s.bus.Publish(ctx, events.Envelope{
		EventID:       uuid.NewString(),
		EventType:     "PostCreated",
		OccurredAt:    time.Now().UTC(),
		AggregateType: "post",
		AggregateID:   uintToString(post.ID),
		ActorUserID:   uintToString(uid),
		Version:       1,
		Payload:       payload,
	}); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) Update(ctx context.Context, slug string, uid uint, title, text *string) (*models.Post, error) {
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

	post, err := s.repo.UpdateOwnedBy(ctx, slug, uid, updates)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	return post, nil
}

func (s *PostService) Delete(ctx context.Context, slug string, uid uint) error {
	err := s.repo.DeleteOwnedBy(ctx, slug, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	return nil
}

func (s *PostService) Get(ctx context.Context, slug string) (*models.Post, error) {
	post, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return post, nil
}

func (s *PostService) List(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error) {
	return s.repo.List(ctx, page, limit, q)
}

func uintToString(v uint) string {
	// не идеально, но ок для старта. Потом приведём к нормальному виду под твои модели/ID
	return fmt.Sprint(v)
}
