package repositories

import (
	"context"
	"go_blog/models"
	"go_blog/testhelpers"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostRepository_GetBySlug_CacheHit(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)
	rdb := testhelpers.SetupTestRedis(t)

	repo := NewPostRepository(tx, rdb)

	user := &models.User{
		Nickname: "u",
		Email:    "cache@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	post := &models.Post{
		Title:    "Cached",
		Slug:     "cached",
		UserID:   user.ID,
		IsActive: true,
	}
	require.NoError(t, tx.Create(post).Error)

	resp1, err := repo.GetBySlug(context.Background(), "cached")
	require.NoError(t, err)
	require.Equal(t, "Cached", resp1.Title)

	// удаляем из БД, но кэш должен отдать старое
	require.NoError(t, tx.Unscoped().Delete(&models.Post{}, post.ID).Error)

	resp2, err := repo.GetBySlug(context.Background(), "cached")
	require.NoError(t, err)
	require.Equal(t, "Cached", resp2.Title)
}

func TestPostRepository_GetBySlug_CacheInvalidatedOnUpdate(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)
	rdb := testhelpers.SetupTestRedis(t)

	repo := NewPostRepository(tx, rdb)

	user := &models.User{
		Nickname: "u",
		Email:    "upd@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	post := &models.Post{
		Title:    "Old",
		Slug:     "upd",
		UserID:   user.ID,
		IsActive: true,
	}
	require.NoError(t, tx.Create(post).Error)

	resp1, err := repo.GetBySlug(context.Background(), "upd")
	require.NoError(t, err)
	require.Equal(t, "Old", resp1.Title)

	// важно: ключ "title" (как в твоём repo updates map)
	_, err = repo.UpdateOwnedBy(context.Background(), "upd", user.ID, map[string]any{"title": "New"})
	require.NoError(t, err)

	resp2, err := repo.GetBySlug(context.Background(), "upd")
	require.NoError(t, err)
	require.Equal(t, "New", resp2.Title)
}

func TestPostRepository_List_CacheVersioned(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)
	rdb := testhelpers.SetupTestRedis(t)

	repo := NewPostRepository(tx, rdb)

	user := &models.User{
		Nickname: "u",
		Email:    "listcache@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	require.NoError(t, tx.Create(&models.Post{
		Title:    "One",
		Slug:     "one",
		UserID:   user.ID,
		IsActive: true,
	}).Error)

	posts1, total1, err := repo.List(context.Background(), 1, 10, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total1)
	require.Len(t, posts1, 1)

	_, err = repo.Create(context.Background(), user.ID, "Two", "text")
	require.NoError(t, err)

	posts2, total2, err := repo.List(context.Background(), 1, 10, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), total2)
	require.Len(t, posts2, 2)
}
