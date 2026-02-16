package repositories_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"testing"
)

func TestUserRepository_Duplicate(t *testing.T) {
	deps := setupTestEnv(t)
	defer deps.Clean()

	repo := repositories.NewUserRepository(deps.DB)
	ctx := context.Background()

	user1 := &models.User{
		Nickname: "original_user",
		Email:    "dup@test.com",
		Password: "hash",
		Role:     "user",
		IsActive: true,
	}

	err := repo.CreateTx(ctx, deps.DB, user1)
	require.NoError(t, err)

	user2 := &models.User{
		Nickname: "imposter",
		Email:    "dup@test.com", // <--- Дубликат
		Password: "hash",
		Role:     "user",
		IsActive: true,
	}

	err = repo.CreateTx(ctx, deps.DB, user2)
	require.Error(t, err)
	assert.Equal(t, repositories.ErrUserExists, err)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	deps := setupTestEnv(t)
	defer deps.Clean()

	repo := repositories.NewUserRepository(deps.DB)
	ctx := context.Background()

	email := "findme@test.com"
	user := &models.User{
		Nickname: "finder",
		Email:    email,
		Password: "hash",
		IsActive: true, // Важно! В тебя Where("is_active = ?", true)
	}

	err := repo.CreateTx(ctx, deps.DB, user)
	require.NoError(t, err)

	found, err := repo.FindByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, email, found.Email)
}
