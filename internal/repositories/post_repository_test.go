package repositories_test

import (
	"context"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	testredis "github.com/testcontainers/testcontainers-go/modules/redis" // Модуль для Redis
	"github.com/testcontainers/testcontainers-go/wait"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Структура для хранения зависимостей теста
type TestDeps struct {
	DB    *gorm.DB
	RDB   *redis.Client
	Clean func() // Функция для очистки контейнеров
}

// setupTestEnv поднимает Postgres и Redis в Docker
// Это запускается один раз перед тестами или перед каждым тестом (как настроим).
func setupTestEnv(t *testing.T) *TestDeps {
	ctx := context.Background()

	// --- 1. POSTGRES ---
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	require.NoError(t, err, "failed to start postgres container")

	// --- 2. REDIS ---
	redisContainer, err := testredis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(10*time.Second),
		),
	)
	require.NoError(t, err, "failed to start redis container")

	// --- 3. ПОДКЛЮЧЕНИЕ К POSTGRES ---
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(postgresDriver.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Миграции
	err = db.AutoMigrate(&models.User{}, &models.Post{})
	require.NoError(t, err)

	// --- 4. ПОДКЛЮЧЕНИЕ К REDIS ---
	redisURI, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// testcontainers возвращает строку вида "redis://localhost:port",
	// go-redis умеет её парсить через ParseURL
	opt, err := redis.ParseURL(redisURI)
	require.NoError(t, err)

	rdb := redis.NewClient(opt)

	// --- 5. CLEANUP FUNCTION ---
	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres: %s", err)
		}
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis: %s", err)
		}
	}

	return &TestDeps{
		DB:    db,
		RDB:   rdb,
		Clean: cleanup,
	}
}

func TestPostRepository_FullCycle(t *testing.T) {
	// Поднимаем окружение
	deps := setupTestEnv(t)
	defer deps.Clean() // Убьем контейнеры в конце теста

	// Инициализируем репозиторий с DB и Redis
	repo := repositories.NewPostRepository(deps.DB, deps.RDB)
	ctx := context.Background()

	// --- СОЗДАЕМ ЮЗЕРА ---
	user := &models.User{
		Nickname: "test_integration_user",
		Email:    "test_integration@example.com",
		Password: "hashed_password_123", // Поле not null, надо заполнить
		Role:     "user",
		IsActive: true,
	}
	// Создаем юзера напрямую через GORM, минуя репозитории (нам просто нужна запись в БД)
	err := deps.DB.Create(user).Error
	require.NoError(t, err, "failed to create test user")

	// ЗАПОМИНАЕМ ЕГО ID!
	// GORM автоматически заполнил user.ID после создания
	uid := user.ID

	// Данные для теста
	var createdPost *models.Post
	title := "Test Title Integration"
	text := "Test Text Content"

	// --- ШАГ 1: CREATE ---
	t.Run("CreateTx", func(t *testing.T) {
		// Используем uid, который мы получили выше
		post, err := repo.CreateTx(ctx, deps.DB, uid, title, text)

		require.NoError(t, err)
		require.NotNil(t, post)

		assert.Equal(t, uid, post.UserID) // Проверяем, что пост привязался к юзеру
		assert.Equal(t, title, post.Title)
		assert.NotEmpty(t, post.Slug)

		createdPost = post
	})

	// --- ШАГ 2: GET (Проверяем чтение) ---
	t.Run("GetBySlug", func(t *testing.T) {
		require.NotNil(t, createdPost)

		fetched, err := repo.GetBySlug(ctx, createdPost.Slug)
		require.NoError(t, err)
		assert.Equal(t, createdPost.ID, fetched.ID)
		assert.Equal(t, createdPost.Title, fetched.Title)
	})

	// --- ШАГ 3: UPDATE ---
	t.Run("UpdateTx", func(t *testing.T) {
		require.NotNil(t, createdPost)

		newTitle := "Updated Title"
		updates := map[string]any{
			"title": newTitle,
		}

		updated, err := repo.UpdateTx(ctx, deps.DB, createdPost.Slug, uid, updates)
		require.NoError(t, err)
		assert.Equal(t, newTitle, updated.Title)

		// Проверяем через Get, что в базе обновилось
		fetched, _ := repo.GetBySlug(ctx, createdPost.Slug)
		assert.Equal(t, newTitle, fetched.Title)
	})

	// --- ШАГ 4: DELETE ---
	t.Run("DeleteTx", func(t *testing.T) {
		require.NotNil(t, createdPost)

		// Исправлено: DeleteTx возвращает (*models.Post, error)
		deletedPost, err := repo.DeleteTx(ctx, deps.DB, createdPost.Slug, uid)
		require.NoError(t, err)
		require.NotNil(t, deletedPost)

		// Проверяем, что ID удаленного поста совпадает
		assert.Equal(t, createdPost.ID, deletedPost.ID)

		// Проверяем, что в базе его больше нет (или он помечен как удаленный)
		_, errGet := repo.GetBySlug(ctx, createdPost.Slug)
		assert.Error(t, errGet) // Ожидаем ошибку (RecordNotFound)
	})
}
