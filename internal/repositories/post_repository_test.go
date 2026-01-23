package repositories

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"go_blog/models"
	"go_blog/testhelpers"
	"testing"
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

	resp, err := repo.GetBySlug(context.Background(), "title")
	require.NoError(t, err)
	require.Equal(t, "Title", resp.Title)
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
		IsActive: false,
	}
	require.NoError(t, tx.Create(post).Error)

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

	out, err := repo.List(context.Background(), 1, 10, "")
	require.NoError(t, err)

	require.Equal(t, int64(5), out.Total)
	require.Len(t, out.Posts, 5)
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

	out, err := repo.List(context.Background(), 1, 10, "go")
	require.NoError(t, err)

	require.Equal(t, int64(1), out.Total)
	require.Equal(t, "Go tutorial", out.Posts[0].Title)

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
		map[string]any{"title": "New"},
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
}
