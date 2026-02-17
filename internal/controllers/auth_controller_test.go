package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go_blog/internal/controllers"
	"go_blog/internal/dto"
	"go_blog/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- 1. MOCK SERVICE ---
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req dto.RegisterRequest, ua, ip string) (*dto.TokenPairResponse, error) {
	args := m.Called(ctx, req, ua, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TokenPairResponse), args.Error(1)
}

// Заглушки для остальных методов интерфейса (чтобы Go не ругался)
func (m *MockAuthService) Login(ctx context.Context, req dto.LoginRequest, ua, ip string) (*dto.TokenPairResponse, error) {
	args := m.Called(ctx, req, ua, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TokenPairResponse), args.Error(1)
}
func (m *MockAuthService) Refresh(ctx context.Context, rt string, ua, ip string) (*dto.TokenPairResponse, error) {
	args := m.Called(ctx, rt, ua, ip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TokenPairResponse), args.Error(1)
}
func (m *MockAuthService) Logout(ctx context.Context, rt string) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

// --- 2. TESTS ---

func TestAuthController_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success Registration", func(t *testing.T) {
		// A. Setup
		mockService := new(MockAuthService)
		controller := controllers.NewAuthController(mockService)

		r := gin.New()
		r.POST("/register", controller.Register)

		// Данные запроса
		reqBody := dto.RegisterRequest{
			Nickname: "tester",
			Email:    "test@test.com",
			Password: "strongpassword",
		}

		// Ожидаемый ответ от сервиса
		expectedTokens := &dto.TokenPairResponse{
			AccessToken:  "access.token",
			RefreshToken: "refresh.token",
		}

		// B. Mock Expectations
		// Ожидаем, что контроллер вызовет сервис с нашими данными
		mockService.On("Register", mock.Anything, reqBody, "", "").
			Return(expectedTokens, nil)

		// C. Request Execution
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// D. Assertions
		assert.Equal(t, http.StatusCreated, w.Code) // 201 Created
		assert.Contains(t, w.Body.String(), "access.token")

		mockService.AssertExpectations(t)
	})

	t.Run("Validation Error (Bad JSON)", func(t *testing.T) {
		mockService := new(MockAuthService)
		controller := controllers.NewAuthController(mockService)
		r := gin.New()
		r.POST("/register", controller.Register)

		// Шлем пустой JSON, а поля обязательные
		reqBody := dto.RegisterRequest{}

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Сервис даже не должен вызываться, валидация отработает в контроллере
		assert.NotEqual(t, http.StatusOK, w.Code)
		// Проверяем, что вернулась ошибка валидации (структура ответа)
		assert.Contains(t, w.Body.String(), "failed validation")

		mockService.AssertNotCalled(t, "Register")
	})

	t.Run("Service Error (User Exists)", func(t *testing.T) {
		mockService := new(MockAuthService)
		controller := controllers.NewAuthController(mockService)
		r := gin.New()
		r.POST("/register", controller.Register)

		reqBody := dto.RegisterRequest{
			Nickname: "tester",
			Email:    "busy@test.com",
			Password: "123",
		}

		mockService.On("Register", mock.Anything, reqBody, "", "").
			Return(nil, services.ErrUserExists)

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Проверяем, что контроллер отловил services.ErrUserExists и вернул 400
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "user already exists")
	})
}
