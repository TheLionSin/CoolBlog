package repositories_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"testing"
	"time"
)

func TestOutboxRepository_FetchBatch(t *testing.T) {
	deps := setupTestEnv(t)
	defer deps.Clean()

	require.NoError(t, deps.DB.AutoMigrate(&models.OutboxEvent{}))
	repo := repositories.NewOutboxRepository(deps.DB)
	ctx := context.Background()

	// 1. Создаем 3 события
	events := []models.OutboxEvent{
		{EventID: uuid.NewString(), Topic: "t1", Status: models.OutboxNew, Payload: "1"},
		{EventID: uuid.NewString(), Topic: "t2", Status: models.OutboxNew, Payload: "2"},
		{EventID: uuid.NewString(), Topic: "t3", Status: models.OutboxSent, Payload: "3"}, // SENT не должен попасть
	}

	// ВАЖНО: Проверяем ошибку при создании
	err := deps.DB.Create(&events).Error
	require.NoError(t, err, "failed to seed database")

	// 2. Пытаемся взять пачку из 2 штук
	batch, err := repo.FetchBatchForPublish(ctx, 2)
	require.NoError(t, err, "fetch batch failed")

	// 3. Проверки
	assert.Len(t, batch, 2)
	if len(batch) > 0 {
		assert.Equal(t, models.OutboxNew, batch[0].Status)
	}
}

func TestOutboxRepository_Concurrency(t *testing.T) {
	deps := setupTestEnv(t)
	defer deps.Clean()
	require.NoError(t, deps.DB.AutoMigrate(&models.OutboxEvent{}))
	repo := repositories.NewOutboxRepository(deps.DB)
	ctx := context.Background()

	// --- 1. SEED DATA (Заполняем базу) ---
	for i := 0; i < 10; i++ {
		evt := models.OutboxEvent{
			EventID: uuid.NewString(),
			Topic:   "race",
			Status:  models.OutboxNew,
			Payload: "{}",
		}
		// Проверяем каждую вставку!
		err := deps.DB.Create(&evt).Error
		require.NoError(t, err, "failed to insert event %d", i)
	}

	// --- 2. VERIFY DATA EXISTENCE (Проверка перед гонкой) ---
	var count int64
	deps.DB.Model(&models.OutboxEvent{}).Where("status = ?", models.OutboxNew).Count(&count)
	require.Equal(t, int64(10), count, "Database should have 10 new events before starting workers")

	// --- 3. START RACE (Гонка) ---
	results := make(chan []uint, 2)
	errorsChan := make(chan error, 2) // Канал для ошибок воркеров

	// Воркер 1
	go func() {
		batch, err := repo.FetchBatchForPublish(ctx, 5)
		if err != nil {
			errorsChan <- err
			results <- nil
			return
		}
		ids := make([]uint, len(batch))
		for i, item := range batch {
			ids[i] = item.ID
		}
		results <- ids
		errorsChan <- nil
	}()

	// Воркер 2
	go func() {
		time.Sleep(50 * time.Millisecond) // Чуть больше задержка для уверенности
		batch, err := repo.FetchBatchForPublish(ctx, 5)
		if err != nil {
			errorsChan <- err
			results <- nil
			return
		}
		ids := make([]uint, len(batch))
		for i, item := range batch {
			ids[i] = item.ID
		}
		results <- ids
		errorsChan <- nil
	}()

	// Читаем результаты
	ids1 := <-results
	err1 := <-errorsChan

	ids2 := <-results
	err2 := <-errorsChan

	// --- 4. CHECK ERRORS ---
	require.NoError(t, err1, "Worker 1 failed")
	require.NoError(t, err2, "Worker 2 failed")

	// --- 5. ASSERTIONS ---
	t.Logf("Worker 1 got %d items: %v", len(ids1), ids1)
	t.Logf("Worker 2 got %d items: %v", len(ids2), ids2)

	assert.Len(t, ids1, 5, "Worker 1 should get 5 items")
	assert.Len(t, ids2, 5, "Worker 2 should get 5 items")

	// Проверяем пересечения
	uniqueMap := make(map[uint]bool)
	for _, id := range ids1 {
		uniqueMap[id] = true
	}

	for _, id := range ids2 {
		if uniqueMap[id] {
			t.Errorf("COLLISION! ID %d was processed by both workers!", id)
		}
	}
}
