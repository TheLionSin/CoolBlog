package middleware_test

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go_blog/internal/middleware"
	"go_blog/internal/utils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireAuth(t *testing.T) {
	// 1.Задаем секрет для этого теста.
	// t.Setenv работает только на время теста и потом возвращает всё как было.
	testSecret := "my-super-secret-key-for-tests"
	t.Setenv("JWT_SECRET", testSecret)

	gin.SetMode(gin.TestMode)

	// Настраиваем роутер
	r := gin.New()
	r.Use(middleware.RequireAuth())
	r.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("userID")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{"uid": uid, "role": role})
	})

	// 2. Генерация тестовых данных

	// А) Валидный токен (генерим утилитой, она подхватит testSecret из env)
	validToken, err := utils.GenerateAccessJWT(10, "admin")
	require.NoError(t, err)

	// Б) Протухший токен
	// Утилита не умеет делать протухшие токены (там hardcode time.Now()),
	// поэтому тут генерим ручками, ИСПОЛЬЗУЯ ТОТ ЖЕ testSecret
	expiredClaims := jwt.MapClaims{
		"sub":  10,
		"role": "user",
		"exp":  time.Now().Add(-1 * time.Hour).Unix(), // Протух час назад
	}
	// Подписываем нашим тестовым секретом
	expiredToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).
		SignedString([]byte(testSecret))

	// 3. ТАБЛИЧНЫЙ ТЕСТ (Table-Driven)
	tests := []struct {
		name         string
		header       string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Success Login",
			header:       "Bearer " + validToken,
			expectedCode: http.StatusOK,
			expectedBody: `"uid":10`, // Проверяем, что ID распарсился
		},
		{
			name:         "No Header",
			header:       "",
			expectedCode: http.StatusUnauthorized,
			expectedBody: "missing bearer token",
		},
		{
			name:         "Invalid Format",
			header:       validToken, // Забыли Bearer
			expectedCode: http.StatusUnauthorized,
			expectedBody: "missing bearer token",
		},
		{
			name:         "Garbage Token",
			header:       "Bearer blablabla",
			expectedCode: http.StatusUnauthorized,
			expectedBody: "invalid token",
		},
		{
			name:         "Expired Token",
			header:       "Bearer " + expiredToken,
			expectedCode: http.StatusUnauthorized,
			expectedBody: "invalid token", // Обычно библиотека пишет "token is expired" или просто invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}
