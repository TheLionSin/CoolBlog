package routes

import (
	"go_blog/internal/controllers"
	"go_blog/internal/services"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPostRoutes(r *gin.Engine,
	postService *services.PostService,
	commentService *services.CommentService,
	likeService *services.LikeService) {

	// 1. ИНИЦИАЛИЗАЦИЯ: Создаем экземпляр нашего нового контроллера
	postController := controllers.NewPostController(postService)
	commentController := controllers.NewCommentController(commentService)
	likeController := controllers.NewLikeController(likeService)

	// --- Публичные роуты ---

	// Используем методы экземпляра (.List, .Get), а не вызов функции
	r.GET("/posts", postController.List)
	r.GET("/posts/:slug", postController.Get)

	r.GET("/posts/:slug/comments", commentController.List)
	r.GET("/posts/:slug/likes", likeController.GetLikes)

	// --- Приватные роуты ---
	auth := r.Group("/posts")
	auth.Use(middleware.RequireAuth())

	// Посты: используем методы нового контроллера
	auth.POST("", postController.Create)
	auth.PUT("/:slug", postController.Update)
	auth.DELETE("/:slug", postController.Delete)

	auth.POST("/:slug/like", likeController.Like)
	auth.DELETE("/:slug/like", likeController.Unlike)

	auth.POST("/:slug/comments", commentController.Create)
	auth.DELETE("/comments/:id", commentController.Delete)
}
