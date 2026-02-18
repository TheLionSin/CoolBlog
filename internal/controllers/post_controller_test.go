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
	"go_blog/internal/models"
	"go_blog/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- MOCK ---
type MockPostService struct {
	mock.Mock
}

func (m *MockPostService) Create(ctx context.Context, uid uint, title, text string) (*models.Post, error) {
	args := m.Called(ctx, uid, title, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockPostService) Get(ctx context.Context, slug string) (*models.Post, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockPostService) Update(ctx context.Context, slug string, uid uint, title, text *string) (*models.Post, error) {
	args := m.Called(ctx, slug, uid, title, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Post), args.Error(1)
}

func (m *MockPostService) Delete(ctx context.Context, slug string, uid uint) error {
	args := m.Called(ctx, slug, uid)
	return args.Error(0)
}

func (m *MockPostService) List(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error) {
	args := m.Called(ctx, page, limit, q)
	return args.Get(0).([]models.Post), args.Get(1).(int64), args.Error(2)
}

// --- SETUP HELPER ---
func setupPostController(t *testing.T) (*gin.Engine, *MockPostService, *controllers.PostController) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockPostService)
	controller := controllers.NewPostController(mockService)

	r := gin.New()
	// Эмулируем Auth Middleware: всегда UserID=10
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(10))
	})

	return r, mockService, controller
}

// --- TESTS ---

func TestPostController_Create(t *testing.T) {
	r, mockService, controller := setupPostController(t)
	r.POST("/posts", controller.Create)

	t.Run("Success Create", func(t *testing.T) {
		reqBody := dto.PostCreateRequest{Title: "New Post", Text: "Content"}
		expectedPost := &models.Post{Title: "New Post", Slug: "new-post", UserID: 10}

		mockService.On("Create", mock.Anything, uint(10), reqBody.Title, reqBody.Text).
			Return(expectedPost, nil)

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "new-post")
	})

	t.Run("Validation Error", func(t *testing.T) {
		reqBody := dto.PostCreateRequest{Title: "", Text: "Content"} // Пустой Title
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		// Сервис НЕ должен вызываться
		mockService.AssertNotCalled(t, "Create")
	})
}

func TestPostController_Get(t *testing.T) {
	r, mockService, controller := setupPostController(t)
	r.GET("/posts/:slug", controller.Get)

	t.Run("Success", func(t *testing.T) {
		slug := "my-post"
		mockService.On("Get", mock.Anything, slug).
			Return(&models.Post{Slug: slug, Title: "title"}, nil)
		req := httptest.NewRequest(http.MethodGet, "/posts/"+slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), slug)
	})

	t.Run("Not found", func(t *testing.T) {
		slug := "unknown"
		mockService.On("Get", mock.Anything, slug).
			Return(nil, services.ErrPostNotFound)

		req := httptest.NewRequest(http.MethodGet, "/posts/"+slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestPostController_Update(t *testing.T) {
	r, mockService, controller := setupPostController(t)
	r.PUT("/posts/:slug", controller.Update)

	slug := "test-slug"

	t.Run("Success Update", func(t *testing.T) {
		newTitle := "Updated title"
		reqBody := dto.PostUpdateRequest{Title: &newTitle}
		var noTextUpdate *string = nil
		isCorrectTitle := mock.MatchedBy(func(s *string) bool {
			return s != nil && *s == newTitle
		})
		mockService.On("Update",
			mock.Anything,  // ctx
			slug,           // slug
			uint(10),       // userID
			isCorrectTitle, // заголовок
			noTextUpdate,   // текст (nil)
		).Return(&models.Post{Slug: slug, Title: newTitle}, nil)

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/posts/"+slug, bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), newTitle)
	})

	t.Run("No Fields To Update", func(t *testing.T) {
		// Шлем пустой JSON {}
		reqBody := dto.PostUpdateRequest{}

		// Сервис возвращает ошибку ErrNoFieldsToUpdate
		mockService.On("Update", mock.Anything, slug, uint(10), (*string)(nil), (*string)(nil)).
			Return(nil, services.ErrNoFieldsToUpdate)

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/posts/"+slug, bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "no fields to update")
	})

	t.Run("Not found (Update)", func(t *testing.T) {
		newTitle := "T"
		reqBody := dto.PostUpdateRequest{Title: &newTitle}

		mockService.On("Update", mock.Anything, slug, uint(10), mock.Anything, mock.Anything).
			Return(nil, services.ErrPostNotFound)
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/posts/"+slug, bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestPostController_Delete(t *testing.T) {
	slug := "to-delete"

	t.Run("Success Delete", func(t *testing.T) {
		// 1. Создаем НОВЫЙ мир для этого теста
		r, mockService, controller := setupPostController(t)
		r.DELETE("/posts/:slug", controller.Delete)

		// 2. Настраиваем
		mockService.On("Delete", mock.Anything, slug, uint(10)).
			Return(nil)

		// 3. Проверяем
		req := httptest.NewRequest(http.MethodDelete, "/posts/"+slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Not found (Delete)", func(t *testing.T) {
		// 1. Опять создаем НОВЫЙ мир (чистый мок)
		r, mockService, controller := setupPostController(t)
		r.DELETE("/posts/:slug", controller.Delete)

		// 2. Теперь мок знает только про ошибку
		mockService.On("Delete", mock.Anything, slug, uint(10)).
			Return(services.ErrPostNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/posts/"+slug, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestPostController_List(t *testing.T) {
	// То же самое для List - убираем общий setup

	t.Run("Success List With Pagination", func(t *testing.T) {
		// Новый сетап
		r, mockService, controller := setupPostController(t)
		r.GET("/posts", controller.List)

		mockService.On("List", mock.Anything, 2, 5, "golang").
			Return([]models.Post{{Title: "Post 1"}, {Title: "Post 2"}}, int64(100), nil)

		req := httptest.NewRequest(http.MethodGet, "/posts?page=2&limit=5&q=golang", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Post 1")
		assert.Contains(t, w.Body.String(), `"total":100`)
	})

	t.Run("Service Error", func(t *testing.T) {
		// Новый сетап
		r, mockService, controller := setupPostController(t)
		r.GET("/posts", controller.List)

		mockService.On("List", mock.Anything, 1, 10, "").
			Return([]models.Post{}, int64(0), services.ErrToken)

		req := httptest.NewRequest(http.MethodGet, "/posts", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
