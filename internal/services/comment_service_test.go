package services_test

import (
	"context"
	"errors"
	"go_blog/internal/models"
	"go_blog/internal/services"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// --- MOCKS ---

type MockCommentRepo struct {
	mock.Mock
}

func (m *MockCommentRepo) CreateTx(ctx context.Context, tx *gorm.DB, comment *models.Comment) error {
	args := m.Called(ctx, mock.Anything, comment)
	if args.Error(0) == nil {
		comment.ID = 1 // Эмулируем присвоение ID базой
	}
	return args.Error(0)
}

func (m *MockCommentRepo) GetByID(ctx context.Context, id uint) (*models.Comment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Comment), args.Error(1)
}

func (m *MockCommentRepo) DeleteTx(ctx context.Context, tx *gorm.DB, comment *models.Comment) error {
	args := m.Called(ctx, mock.Anything, comment)
	return args.Error(0)
}

func (m *MockCommentRepo) ListByPostID(ctx context.Context, postID uint) ([]models.Comment, error) {
	args := m.Called(ctx, postID)
	return args.Get(0).([]models.Comment), args.Error(1)
}

// Повторяем мок для PostRepo (нужен только метод GetBySlug)
type MockPostRepoForComment struct {
	mock.Mock
}

func (m *MockPostRepoForComment) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

// Повторяем мок для Outbox
type MockOutboxRepoForComment struct {
	mock.Mock
}

func (m *MockOutboxRepoForComment) CreateTx(ctx context.Context, tx *gorm.DB, evt *models.OutboxEvent) error {
	args := m.Called(ctx, mock.Anything, evt)
	return args.Error(0)
}

// --- SETUP ---
func setupCommentServiceTest(t *testing.T) (*services.CommentService, *MockCommentRepo, *MockPostRepoForComment, *MockOutboxRepoForComment) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	mComment := new(MockCommentRepo)
	mPost := new(MockPostRepoForComment)
	mOutbox := new(MockOutboxRepoForComment)

	service := services.NewCommentService(db, mComment, mPost, mOutbox)

	return service, mComment, mPost, mOutbox
}

// --- TESTS ---

func TestCommentService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success Create with Outbox Event", func(t *testing.T) {
		service, mComment, mPost, mOutbox := setupCommentServiceTest(t)

		postSlug := "test-post"
		userID := uint(5)
		text := "Hello World"

		// 1. Ожидаем, что сервис пойдет искать пост по слагу
		mPost.On("GetBySlug", ctx, postSlug).
			Return(&models.Post{Model: gorm.Model{ID: 10}, UserID: 1, Slug: postSlug}, nil)

		// 2. Ожидаем создание коммента в транзакции
		mComment.On("CreateTx", ctx, mock.Anything, mock.MatchedBy(func(c *models.Comment) bool {
			return c.Text == text && c.PostID == 10 && c.UserID == userID
		})).Return(nil)

		// 3. Ожидаем создание события в Outbox
		mOutbox.On("CreateTx", ctx, mock.Anything, mock.MatchedBy(func(evt *models.OutboxEvent) bool {
			return evt.EventType == "CommentCreated" && evt.AggregateID == "1"
		})).Return(nil)

		result, err := service.Create(ctx, postSlug, userID, text)

		require.NoError(t, err)
		assert.Equal(t, uint(1), result.ID)
		assert.Equal(t, text, result.Text)

		mPost.AssertExpectations(t)
		mComment.AssertExpectations(t)
		mOutbox.AssertExpectations(t)
	})

	t.Run("Post Not Found", func(t *testing.T) {
		service, mComment, mPost, mOutbox := setupCommentServiceTest(t)

		mPost.On("GetBySlug", ctx, "unknown").
			Return(nil, errors.New("post not found"))

		result, err := service.Create(ctx, "unknown", 1, "text")

		assert.Error(t, err)
		assert.Nil(t, result)

		mComment.AssertNotCalled(t, "CreateTx")
		mOutbox.AssertNotCalled(t, "CreateTx")
	})
}
