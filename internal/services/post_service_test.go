package services_test

import (
	"context"
	"errors"
	"go_blog/internal/models"
	"go_blog/internal/services"
	"testing"

	"github.com/glebarez/sqlite" // <--- Это чистый Go, работает везде
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// --- 1. MOCKS (Заглушки) ---
// Мы создаем фейковые репозитории, которые ничего не пишут в базу,
// а просто возвращают то, что мы им скажем.

type MockPostRepo struct {
	mock.Mock
}

// Заглушка для CreateTx
func (m *MockPostRepo) CreateTx(ctx context.Context, tx *gorm.DB, uid uint, title, text string) (*models.Post, error) {
	// mock.Anything означает, что нам плевать, какой именно *gorm.DB пришел, главное что пришел
	args := m.Called(ctx, mock.Anything, uid, title, text)

	// Возвращаем (Post, error)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

// Заглушки для остальных методов (нужны, чтобы соответствовать интерфейсу)
func (m *MockPostRepo) UpdateTx(ctx context.Context, tx *gorm.DB, slug string, uid uint, updates map[string]any) (*models.Post, error) {
	args := m.Called(ctx, mock.Anything, slug, uid, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}
func (m *MockPostRepo) DeleteTx(ctx context.Context, tx *gorm.DB, slug string, uid uint) (*models.Post, error) {
	args := m.Called(ctx, mock.Anything, slug, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}
func (m *MockPostRepo) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}
func (m *MockPostRepo) List(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error) {
	args := m.Called(ctx, page, limit, q)
	return args.Get(0).([]models.Post), args.Get(1).(int64), args.Error(2)
}

type MockOutboxRepo struct {
	mock.Mock
}

func (m *MockOutboxRepo) CreateTx(ctx context.Context, tx *gorm.DB, evt *models.OutboxEvent) error {
	args := m.Called(ctx, mock.Anything, evt)
	return args.Error(0)
}

// --- 2. SETUP (Настройка теста) ---

func setupServiceTest(t *testing.T) (*services.PostService, *MockPostRepo, *MockOutboxRepo) {
	// Поднимаем SQLite в оперативной памяти (супер быстро)
	// Это нужно, чтобы s.db.Transaction работала реально
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	mockRepo := new(MockPostRepo)
	mockOutbox := new(MockOutboxRepo)

	// ВАЖНО: Твой PostService должен принимать интерфейсы, как мы договаривались
	service := services.NewPostService(db, mockRepo, mockOutbox)

	return service, mockRepo, mockOutbox
}

// --- 3. ТЕСТЫ ---

func TestPostService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("Success: Post and Event Created", func(t *testing.T) {
		service, mockRepo, mockOutbox := setupServiceTest(t)

		// Данные
		uid := uint(1)
		title := "New Post"
		text := "Content"
		expectedPost := &models.Post{
			Model:  gorm.Model{ID: 1},
			UserID: uid, Title: title, Slug: "new-post"}

		// НАСТРОЙКА МОКОВ (Сценарий)
		// 1. Ожидаем вызов репозитория. Возвращаем успешный пост.
		mockRepo.On("CreateTx", ctx, mock.Anything, uid, title, text).
			Return(expectedPost, nil)

		// 2. Ожидаем вызов Outbox. Проверяем, что событие правильное.
		mockOutbox.On("CreateTx", ctx, mock.Anything, mock.MatchedBy(func(evt *models.OutboxEvent) bool {
			return evt.EventType == "PostCreated" && evt.Payload != ""
		})).Return(nil)

		// ВЫПОЛНЕНИЕ
		result, err := service.Create(ctx, uid, title, text)

		// ПРОВЕРКА
		require.NoError(t, err)
		assert.Equal(t, expectedPost.ID, result.ID)

		// Убеждаемся, что моки были вызваны
		mockRepo.AssertExpectations(t)
		mockOutbox.AssertExpectations(t)
	})

	t.Run("Failure: Repo Returns Error", func(t *testing.T) {
		service, mockRepo, mockOutbox := setupServiceTest(t)

		// Сценарий: Репозиторий возвращает ошибку базы
		mockRepo.On("CreateTx", ctx, mock.Anything, uint(1), "Fail", "Text").
			Return(nil, errors.New("db connection lost"))

		// Outbox НЕ должен вызваться (так как транзакция прервется раньше)

		result, err := service.Create(ctx, 1, "Fail", "Text")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "db connection lost")

		mockRepo.AssertExpectations(t)
		mockOutbox.AssertNotCalled(t, "CreateTx") // Гарантируем, что событие не создалось
	})

	t.Run("Failure: Outbox Returns Error", func(t *testing.T) {
		service, mockRepo, mockOutbox := setupServiceTest(t)

		// Сценарий: Пост создался, но Outbox упал
		post := &models.Post{Model: gorm.Model{ID: 1}}
		mockRepo.On("CreateTx", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(post, nil)

		mockOutbox.On("CreateTx", ctx, mock.Anything, mock.Anything).
			Return(errors.New("kafka error"))

		result, err := service.Create(ctx, 1, "Title", "Text")

		// Ожидаем ошибку
		assert.Error(t, err)
		assert.Nil(t, result) // Сервис должен вернуть nil, так как транзакция откатилась!

		mockRepo.AssertExpectations(t)
		mockOutbox.AssertExpectations(t)
	})
}
