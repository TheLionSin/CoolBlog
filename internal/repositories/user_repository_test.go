package repositories

import (
	"context"
	"github.com/stretchr/testify/require"
	"go_blog/models"
	"go_blog/testhelpers"
	"testing"
)

func TestUserRepository_Create_OK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewUserRepository(tx)

	u := &models.User{
		Nickname: "test",
		Email:    "test@test.com",
		Password: "12345s",
	}

	err := repo.Create(context.Background(), u)
	require.NoError(t, err)
	require.NotZero(t, u.ID)
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewUserRepository(tx)

	u1 := &models.User{
		Nickname: "dup",
		Email:    "dup@test.com",
		Password: "12345s",
	}
	u2 := &models.User{
		Nickname: "dup",
		Email:    "dup@test.com",
		Password: "12345s",
	}

	require.NoError(t, repo.Create(context.Background(), u1))

	err := repo.Create(context.Background(), u2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUserExists)
}

func TestUserRepository_FindByEmail_OnlyActive(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewUserRepository(tx)

	active := &models.User{
		Nickname: "active",
		Email:    "active@test.com",
		Password: "12345s",
	}
	inactive := &models.User{
		Nickname: "inactive",
		Email:    "inactive@test.com",
		Password: "12345s",
		IsActive: false,
	}

	require.NoError(t, repo.Create(context.Background(), active))
	require.NoError(t, repo.Create(context.Background(), inactive))

	tx.Model(&models.User{}).Where("id = ?", inactive.ID).Update("is_active", false)

	got, err := repo.FindByEmail(context.Background(), "active@test.com")
	require.NoError(t, err)
	require.Equal(t, "active@test.com", got.Email)

	_, err = repo.FindByEmail(context.Background(), "inactive@test.com")
	require.Error(t, err)
}

func TestUserRepository_FindByID_OK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	tx := testhelpers.BeginTx(t, db)

	repo := NewUserRepository(tx)

	u := &models.User{
		Nickname: "id",
		Email:    "id@test.com",
		Password: "12345s",
	}

	require.NoError(t, repo.Create(context.Background(), u))

	got, err := repo.FindByID(context.Background(), u.ID)
	require.NoError(t, err)
	require.Equal(t, u.Email, got.Email)
}
