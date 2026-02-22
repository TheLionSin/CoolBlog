package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"go_blog/internal/controllers"
	"go_blog/internal/dto"
	"go_blog/internal/models"
	"go_blog/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- MOCK ---
type MockCommentService struct {
	mock.Mock
}

func (m *MockCommentService) Create(ctx context.Context, postSlug string, userID uint, text string) (*models.Comment, error) {
	args := m.Called(ctx, postSlug, userID, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Comment), args.Error(1)
}

func (m *MockCommentService) Delete(ctx context.Context, commentID uint, userID uint) error {
	args := m.Called(ctx, commentID, userID)
	return args.Error(0)
}

func (m *MockCommentService) List(ctx context.Context, postSlug string) ([]dto.CommentResponse, error) {
	args := m.Called(ctx, postSlug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.CommentResponse), args.Error(1)
}

// --- SETUP ---
func setupCommentController(t *testing.T) (*gin.Engine, *MockCommentService, *controllers.CommentController) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockCommentService)
	controller := controllers.NewCommentController(mockService)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(10)) // Эмуляция авторизации
	})

	return r, mockService, controller
}

// --- TESTS ---

func TestCommentController_Create(t *testing.T) {
	slug := "my-post"

	t.Run("Success", func(t *testing.T) {
		r, mockService, controller := setupCommentController(t)
		r.POST("/posts/:slug/comments", controller.Create)

		reqBody := dto.CommentCreateRequest{Text: "Great post!"}
		expectedComment := &models.Comment{Text: "Great post!", PostID: 1, UserID: 10}

		mockService.On("Create", mock.Anything, slug, uint(10), reqBody.Text).
			Return(expectedComment, nil)

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/posts/"+slug+"/comments", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Great post!")
	})

	t.Run("Post Not Found", func(t *testing.T) {
		r, mockService, controller := setupCommentController(t)
		r.POST("/posts/:slug/comments", controller.Create)

		reqBody := dto.CommentCreateRequest{Text: "Great post!"}

		mockService.On("Create", mock.Anything, slug, uint(10), reqBody.Text).
			Return(nil, services.ErrPostNotFound)

		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/posts/"+slug+"/comments", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

}

func TestCommentController_Delete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, mockService, controller := setupCommentController(t)
		r.DELETE("/comments/:id", controller.Delete)

		mockService.On("Delete", mock.Anything, uint(5), uint(10)).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/comments/5", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Forbidden", func(t *testing.T) {
		r, mockService, controller := setupCommentController(t)
		r.DELETE("/comments/:id", controller.Delete)

		mockService.On("Delete", mock.Anything, uint(5), uint(10)).
			Return(services.ErrForbidden) // Пытаемся удалить чужой

		req := httptest.NewRequest(http.MethodDelete, "/comments/5", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestCommentController_List(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, mockService, controller := setupCommentController(t)
		r.GET("/posts/:slug/comments", controller.List)

		slug := "my-post"
		mockResp := []dto.CommentResponse{{ID: 1, Text: "Hello"}}

		mockService.On("List", mock.Anything, slug).Return(mockResp, nil)

		req := httptest.NewRequest(http.MethodGet, "/posts/"+slug+"/comments", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Hello")
	})
}
