package services

import (
	"context"
	"errors"
	"go_blog/dto"
	"go_blog/models"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakePostRepo struct {
	createFn      func(ctx context.Context, uid uint, title, text string) (*models.Post, error)
	updateOwnedFn func(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error)
	deleteOwnedFn func(ctx context.Context, slug string, uid uint) error
	getBySlugFn   func(ctx context.Context, slug string) (dto.PostResponse, error)
	listFn        func(ctx context.Context, page, limit int, q string) (dto.PostListResponse, error)
}

func (f *fakePostRepo) Create(ctx context.Context, uid uint, title, text string) (*models.Post, error) {
	return f.createFn(ctx, uid, title, text)
}
func (f *fakePostRepo) UpdateOwnedBy(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error) {
	return f.updateOwnedFn(ctx, slug, uid, updates)
}
func (f *fakePostRepo) DeleteOwnedBy(ctx context.Context, slug string, uid uint) error {
	return f.deleteOwnedFn(ctx, slug, uid)
}
func (f *fakePostRepo) GetBySlug(ctx context.Context, slug string) (dto.PostResponse, error) {
	return f.getBySlugFn(ctx, slug)
}
func (f *fakePostRepo) List(ctx context.Context, page, limit int, q string) (dto.PostListResponse, error) {
	return f.listFn(ctx, page, limit, q)
}

func TestPostService_Create_Trims(t *testing.T) {
	repo := &fakePostRepo{
		createFn: func(ctx context.Context, uid uint, title, text string) (*models.Post, error) {
			require.Equal(t, uint(10), uid)
			require.Equal(t, "Hello", title)
			require.Equal(t, "World", text)
			return &models.Post{Slug: "hello", Title: title, Text: text, UserID: uid, IsActive: true}, nil
		},
	}
	svc := NewPostService(repo)

	out, err := svc.Create(context.Background(), 10, dto.PostCreateRequest{
		Title: "  Hello  ",
		Text:  "  World ",
	})
	require.NoError(t, err)
	require.Equal(t, "Hello", out.Title)
}

func TestPostService_Update_NoFields(t *testing.T) {
	repo := &fakePostRepo{
		updateOwnedFn: func(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error) {
			t.Fatalf("repo.UpdateOwnedBy must not be called")
			return nil, nil
		},
	}
	svc := NewPostService(repo)

	_, err := svc.Update(context.Background(), "x", 1, dto.PostUpdateRequest{})
	require.ErrorIs(t, err, ErrNoFieldsToUpdate)
}

func TestPostService_Update_TrimsAndMapsNotFound(t *testing.T) {
	title := "  New  "
	text := "  Text "
	repo := &fakePostRepo{
		updateOwnedFn: func(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error) {
			require.Equal(t, "slug", slug)
			require.Equal(t, uint(7), uid)
			require.Equal(t, "New", updates["title"])
			require.Equal(t, "Text", updates["text"])
			return nil, gorm.ErrRecordNotFound
		},
	}
	svc := NewPostService(repo)

	_, err := svc.Update(context.Background(), "slug", 7, dto.PostUpdateRequest{
		Title: &title,
		Text:  &text,
	})
	require.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostService_Update_RepoErrorPassesThrough(t *testing.T) {
	want := errors.New("db down")
	title := "New"
	repo := &fakePostRepo{
		updateOwnedFn: func(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error) {
			return nil, want
		},
	}
	svc := NewPostService(repo)

	_, err := svc.Update(context.Background(), "s", 1, dto.PostUpdateRequest{Title: &title})
	require.ErrorIs(t, err, want)
}

func TestPostService_Delete_MapsNotFound(t *testing.T) {
	repo := &fakePostRepo{
		deleteOwnedFn: func(ctx context.Context, slug string, uid uint) error {
			return gorm.ErrRecordNotFound
		},
	}
	svc := NewPostService(repo)

	err := svc.Delete(context.Background(), "slug", 1)
	require.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostService_Get_MapsNotFound(t *testing.T) {
	repo := &fakePostRepo{
		getBySlugFn: func(ctx context.Context, slug string) (dto.PostResponse, error) {
			return dto.PostResponse{}, gorm.ErrRecordNotFound
		},
	}
	svc := NewPostService(repo)

	_, err := svc.Get(context.Background(), "slug")
	require.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostService_List_PassesThrough(t *testing.T) {
	repo := &fakePostRepo{
		listFn: func(ctx context.Context, page, limit int, q string) (dto.PostListResponse, error) {
			require.Equal(t, 2, page)
			require.Equal(t, 5, limit)
			require.Equal(t, "go", q)
			return dto.PostListResponse{Ok: true, Page: page, Limit: limit, Total: 123}, nil
		},
	}
	svc := NewPostService(repo)

	out, err := svc.List(context.Background(), 2, 5, "go")
	require.NoError(t, err)
	require.Equal(t, int64(123), out.Total)
}
