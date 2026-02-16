package repositories_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"testing"
	"time"
)

func TestTokenRepository_FullCycle(t *testing.T) {
	deps := setupTestEnv(t)
	defer deps.Clean()
	repo := repositories.NewRefreshTokenRepository(deps.DB)
	ctx := context.Background()

	// 1. Нужен User
	user := &models.User{
		Nickname: "token_owner",
		Email:    "token@test.com",
		Password: "hash",
		IsActive: true,
	}
	require.NoError(t, deps.DB.Create(user).Error)

	tokenHash := "some_hashed_string"

	// --- CREATE ---
	token := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		UserAgent: "Mozilla",
		IP:        "127.0.0.1",
	}
	err := repo.CreateTx(ctx, deps.DB, token)
	require.NoError(t, err)
	assert.NotZero(t, token.ID)

	// --- GET BY HASH ---
	fetched, err := repo.GetByHash(ctx, tokenHash)
	require.NoError(t, err)
	assert.Equal(t, token.ID, fetched.ID)
	assert.Equal(t, user.ID, fetched.User.ID) // Проверяем Preload("User")

	// --- DELETE ---
	err = repo.Delete(ctx, token.ID)
	require.NoError(t, err)

	_, err = repo.GetByHash(ctx, tokenHash)
	assert.Error(t, err) // RecordNotFound
}
