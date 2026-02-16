package services_test

import (
	"context"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go_blog/internal/dto"
	"go_blog/internal/models"
	"go_blog/internal/services"
	"gorm.io/gorm"
	"testing"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) CreateTx(ctx context.Context, tx *gorm.DB, user *models.User) error {
	args := m.Called(ctx, mock.Anything, user)
	// Эмулируем присвоение ID базой данных
	if args.Error(0) == nil {
		user.ID = 1
	}
	return args.Error(0)
}

func (m *MockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

type MockTokenRepo struct {
	mock.Mock
}

func (m *MockTokenRepo) CreateTx(ctx context.Context, tx *gorm.DB, token *models.RefreshToken) error {
	args := m.Called(ctx, mock.Anything, token)
	return args.Error(0)
}
func (m *MockTokenRepo) GetByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}
func (m *MockTokenRepo) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockOutboxRepoAuth struct {
	mock.Mock
}

func (m *MockOutboxRepoAuth) CreateTx(ctx context.Context, tx *gorm.DB, evt *models.OutboxEvent) error {
	args := m.Called(ctx, mock.Anything, evt)
	return args.Error(0)
}

// --- SETUP ---

func setupAuthService(t *testing.T) (*services.AuthService, *MockUserRepo, *MockTokenRepo, *MockOutboxRepoAuth) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	mUser := new(MockUserRepo)
	mToken := new(MockTokenRepo)
	mOutbox := new(MockOutboxRepoAuth)

	s := services.NewAuthService(db, mUser, mToken, mOutbox)
	return s, mUser, mToken, mOutbox
}

// --- TESTS ---

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("Success Registration", func(t *testing.T) {
		service, mUser, mToken, mOutbox := setupAuthService(t)

		req := dto.RegisterRequest{
			Nickname: "tester",
			Email:    "test@auth.com",
			Password: "password123",
		}

		// 1. Ожидаем создания Юзера
		mUser.On("CreateTx", ctx, mock.Anything, mock.MatchedBy(func(u *models.User) bool {
			return u.Email == req.Email && u.Nickname == req.Nickname
		})).Return(nil)

		// 2. Ожидаем сохранения Рефреш токена
		mToken.On("CreateTx", ctx, mock.Anything, mock.MatchedBy(func(tok *models.RefreshToken) bool {
			return tok.UserID == 1 && tok.UserAgent == "Mozilla"
		})).Return(nil)

		// 3. Ожидаем событие UserRegistered
		mOutbox.On("CreateTx", ctx, mock.Anything, mock.MatchedBy(func(evt *models.OutboxEvent) bool {
			return evt.EventType == "UserRegistered"
		})).Return(nil)

		// Call
		resp, err := service.Register(ctx, req, "Mozilla", "127.0.0.1")

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)

		mUser.AssertExpectations(t)
		mToken.AssertExpectations(t)
		mOutbox.AssertExpectations(t)
	})
}
