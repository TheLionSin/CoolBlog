package middleware_test

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go_blog/internal/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Поднимаем in-memory Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 2. Подключаем обычный клиент go-redis к нашему фейковому серверу
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limit := 2 // Разрешаем 2 запроса
	window := 1 * time.Second

	// 3. Настраиваем тестовый роутер
	r := gin.New()
	r.Use(middleware.RateLimit(client, limit, window))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	t.Run("Under limit", func(t *testing.T) {
		// Делаем 2 запроса, оба должны пройти (200 OK)
		for i := 0; i < limit; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234" // Эмулируем IP

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("Over limit", func(t *testing.T) {
		// 3-й запрос от того же IP должен упасть (429 Too Many Requests)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "too many requests")
	})

	t.Run("Different IP", func(t *testing.T) {
		// Запрос от ДРУГОГО IP должен пройти успешно (лимиты раздельные)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.5:5678"

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Fail open (Nil Client)", func(t *testing.T) {
		// Создаем новый роутер БЕЗ клиента Redis
		rNil := gin.New()
		rNil.Use(middleware.RateLimit(nil, limit, window))
		rNil.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "success")
		})

		// Запрос должен пройти, даже если "редис упал"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		rNil.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
