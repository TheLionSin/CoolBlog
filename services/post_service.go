package services

import (
	"context"
	"errors"
	"go_blog/dto"
	"go_blog/internal/repositories"
	"go_blog/utils"
	"strings"

	"gorm.io/gorm"
)

type PostService struct {
	repo *repositories.PostRepository
}

func NewPostService(repo *repositories.PostRepository) *PostService {
	return &PostService{repo: repo}
}

func (s *PostService) Create(ctx context.Context, uid uint, req dto.PostCreateRequest) (dto.PostResponse, error) {
	title := strings.TrimSpace(req.Title)
	text := strings.TrimSpace(req.Text)

	post, err := s.repo.Create(ctx, uid, title, text)
	if err != nil {
		return dto.PostResponse{}, err
	}

	return utils.PostToResp(*post), nil
}

func (s *PostService) Update(ctx context.Context, slug string, uid uint, req dto.PostUpdateRequest) (dto.PostResponse, error) {
	updates := map[string]any{}

	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Text != nil {
		updates["text"] = strings.TrimSpace(*req.Text)
	}

	if len(updates) == 0 {
		return dto.PostResponse{}, ErrNoFieldsToUpdate
	}

	post, err := s.repo.UpdateOwnedBy(ctx, slug, uid, updates)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PostResponse{}, ErrPostNotFound
		}
		return dto.PostResponse{}, err
	}

	return utils.PostToResp(*post), nil
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

func (s *PostService) Get(ctx context.Context, slug string) (dto.PostResponse, error) {
	resp, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PostResponse{}, ErrPostNotFound
		}
		return dto.PostResponse{}, err
	}
	return resp, nil
}

func (s *PostService) List(ctx context.Context, page, limit int, q string) (dto.PostListResponse, error) {
	return s.repo.List(ctx, page, limit, q)
}
