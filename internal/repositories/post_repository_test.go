package repositories

import (
	"context"
	"fmt"
	"go_blog/models"
	"go_blog/testhelpers"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPostRepository_Create_OK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	user := &models.User{
		Nickname: "author",
		Email:    "author@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	post, err := repo.Create(context.Background(), user.ID, "Hello world", "text")
	require.NoError(t, err)

	require.NotZero(t, post.ID)
	require.NotEmpty(t, post.Slug)
	require.Equal(t, user.ID, post.UserID)
}

func TestPostRepository_GetBySlug_OK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	user := &models.User{
		Nickname: "u",
		Email:    "u@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	post := &models.Post{
		Title:    "Title",
		Text:     "Text",
		Slug:     "title",
		UserID:   user.ID,
		IsActive: true,
	}
	require.NoError(t, tx.Create(post).Error)

	got, err := repo.GetBySlug(context.Background(), "title")
	require.NoError(t, err)
	require.Equal(t, "Title", got.Title)
	require.Equal(t, "Text", got.Text)
}

func TestPostRepository_GetBySlug_Inactive(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	user := &models.User{
		Nickname: "u",
		Email:    "u2@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	post := &models.Post{
		Title:    "Hidden",
		Slug:     "hidden",
		UserID:   user.ID,
		IsActive: true, // создаём как active
	}
	require.NoError(t, tx.Create(post).Error)

	// теперь гарантированно делаем inactive в БД
	require.NoError(t,
		tx.Model(&models.Post{}).
			Where("id = ?", post.ID).
			Update("is_active", false).Error,
	)

	_, err := repo.GetBySlug(context.Background(), "hidden")
	require.Error(t, err)
}

func TestPostRepository_List_OK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	user := &models.User{
		Nickname: "u",
		Email:    "list@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	for i := 1; i <= 5; i++ {
		p := &models.Post{
			Title:    fmt.Sprintf("Post %d", i),
			Slug:     fmt.Sprintf("post-%d", i),
			UserID:   user.ID,
			IsActive: true,
		}
		require.NoError(t, tx.Create(p).Error)
	}

	posts, total, err := repo.List(context.Background(), 1, 10, "")
	require.NoError(t, err)

	require.Equal(t, int64(5), total)
	require.Len(t, posts, 5)
}

func TestPostRepository_List_Search(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	user := &models.User{
		Nickname: "u",
		Email:    "search@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	require.NoError(t, tx.Create(&models.Post{
		Title:    "Go tutorial",
		Slug:     "go",
		UserID:   user.ID,
		IsActive: true,
	}).Error)

	require.NoError(t, tx.Create(&models.Post{
		Title:    "Python tutorial",
		Slug:     "py",
		UserID:   user.ID,
		IsActive: true,
	}).Error)

	posts, total, err := repo.List(context.Background(), 1, 10, "go")
	require.NoError(t, err)

	require.Equal(t, int64(1), total)
	require.Len(t, posts, 1)
	require.Equal(t, "Go tutorial", posts[0].Title)
}

func TestPostRepository_UpdateOwnedBy_OK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	owner := &models.User{
		Nickname: "o",
		Email:    "o@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(owner).Error)

	post := &models.Post{
		Title:    "Old",
		Slug:     "old",
		UserID:   owner.ID,
		IsActive: true,
	}
	require.NoError(t, tx.Create(post).Error)

	updated, err := repo.UpdateOwnedBy(
		context.Background(),
		"old",
		owner.ID,
		map[string]any{"title": "New"}, // важно: "title" (как в твоём repo updates map)
	)
	require.NoError(t, err)
	require.Equal(t, "New", updated.Title)
}

func TestPostRepository_UpdateOwnedBy_NotOwner(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	owner := &models.User{
		Nickname: "o",
		Email:    "o2@test.com",
		Password: "123",
		IsActive: true,
	}
	other := &models.User{
		Nickname: "x",
		Email:    "x@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(owner).Error)
	require.NoError(t, tx.Create(other).Error)

	post := &models.Post{
		Title:    "Post",
		Slug:     "post",
		UserID:   owner.ID,
		IsActive: true,
	}
	require.NoError(t, tx.Create(post).Error)

	_, err := repo.UpdateOwnedBy(
		context.Background(),
		"post",
		other.ID,
		map[string]any{"title": "Hack"},
	)
	require.Error(t, err)
}

func TestPostRepository_DeleteOwnedBy_OK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewPostRepository(tx, nil)

	user := &models.User{
		Nickname: "u",
		Email:    "del@test.com",
		Password: "123",
		IsActive: true,
	}
	require.NoError(t, tx.Create(user).Error)

	post := &models.Post{
		Title:    "Delete me",
		Slug:     "del",
		UserID:   user.ID,
		IsActive: true,
	}
	require.NoError(t, tx.Create(post).Error)

	err := repo.DeleteOwnedBy(context.Background(), "del", user.ID)
	require.NoError(t, err)

	_, err = repo.GetBySlug(context.Background(), "del")
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
