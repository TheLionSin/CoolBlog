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

	_, err = repo.UpdateOwnedBy(context.Background(), "upd", user.ID, map[string]any{"Title": "New"})
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

	out1, err := repo.List(context.Background(), 1, 10, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), out1.Total)

	_, err = repo.Create(context.Background(), user.ID, "Two", "text")
	require.NoError(t, err)

	out2, err := repo.List(context.Background(), 1, 10, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), out2.Total)

}
