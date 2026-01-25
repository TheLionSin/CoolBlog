package services

import (
	"context"
	"errors"
	"go_blog/internal/events"
	"go_blog/models"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- fake bus (если NewPostService(repo, bus) и Create публикует событие) ----

type fakeBus struct {
	publishFn func(ctx context.Context, e events.Envelope) error
}

func (b *fakeBus) Publish(ctx context.Context, e events.Envelope) error {
	if b.publishFn != nil {
		return b.publishFn(ctx, e)
	}
	return nil
}

// ---- fake repo ----

type fakePostRepo struct {
	createFn      func(ctx context.Context, uid uint, title, text string) (*models.Post, error)
	updateOwnedFn func(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error)
	deleteOwnedFn func(ctx context.Context, slug string, uid uint) error
	getBySlugFn   func(ctx context.Context, slug string) (*models.Post, error)
	listFn        func(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error)
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
func (f *fakePostRepo) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	return f.getBySlugFn(ctx, slug)
}
func (f *fakePostRepo) List(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error) {
	return f.listFn(ctx, page, limit, q)
}

// ---- tests ----

func TestPostService_Create_Trims(t *testing.T) {
	repo := &fakePostRepo{
		createFn: func(ctx context.Context, uid uint, title, text string) (*models.Post, error) {
			require.Equal(t, uint(10), uid)
			require.Equal(t, "Hello", title)
			require.Equal(t, "World", text)
			return &models.Post{Slug: "hello", Title: title, Text: text, UserID: uid, IsActive: true}, nil
		},
	}

	bus := &fakeBus{} // Create публикует — пусть будет no-op
	svc := NewPostService(repo, bus)

	out, err := svc.Create(context.Background(), 10, "  Hello  ", "  World ")
	require.NoError(t, err)
	require.Equal(t, "Hello", out.Title)
	require.Equal(t, "World", out.Text)
}

func TestPostService_Update_NoFields(t *testing.T) {
	repo := &fakePostRepo{
		updateOwnedFn: func(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error) {
			t.Fatalf("repo.UpdateOwnedBy must not be called")
			return nil, nil
		},
	}
	bus := &fakeBus{}
	svc := NewPostService(repo, bus)

	_, err := svc.Update(context.Background(), "x", 1, nil, nil)
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
	bus := &fakeBus{}
	svc := NewPostService(repo, bus)

	_, err := svc.Update(context.Background(), "slug", 7, &title, &text)
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
	bus := &fakeBus{}
	svc := NewPostService(repo, bus)

	_, err := svc.Update(context.Background(), "s", 1, &title, nil)
	require.ErrorIs(t, err, want)
}

func TestPostService_Delete_MapsNotFound(t *testing.T) {
	repo := &fakePostRepo{
		deleteOwnedFn: func(ctx context.Context, slug string, uid uint) error {
			return gorm.ErrRecordNotFound
		},
	}
	bus := &fakeBus{}
	svc := NewPostService(repo, bus)

	err := svc.Delete(context.Background(), "slug", 1)
	require.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostService_Get_MapsNotFound(t *testing.T) {
	repo := &fakePostRepo{
		getBySlugFn: func(ctx context.Context, slug string) (*models.Post, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	bus := &fakeBus{}
	svc := NewPostService(repo, bus)

	_, err := svc.Get(context.Background(), "slug")
	require.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostService_List_PassesThrough(t *testing.T) {
	repo := &fakePostRepo{
		listFn: func(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error) {
			require.Equal(t, 2, page)
			require.Equal(t, 5, limit)
			require.Equal(t, "go", q)
			return []models.Post{
				{Slug: "a", Title: "A"},
				{Slug: "b", Title: "B"},
			}, 123, nil
		},
	}
	bus := &fakeBus{}
	svc := NewPostService(repo, bus)

	posts, total, err := svc.List(context.Background(), 2, 5, "go")
	require.NoError(t, err)
	require.Len(t, posts, 2)
	require.Equal(t, int64(123), total)
}
