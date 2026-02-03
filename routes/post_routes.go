package routes

import (
	"go_blog/internal/controllers"
	"go_blog/internal/repositories"
	"go_blog/internal/services"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPostRoutes(r *gin.Engine,
	postService *services.PostService,
	commentRepo *repositories.CommentRepository,
	likeRepo *repositories.LikeRepository) {

	// 1. ИНИЦИАЛИЗАЦИЯ: Создаем экземпляр нашего нового контроллера
	postController := controllers.NewPostController(postService)

	// --- Публичные роуты ---

	// Используем методы экземпляра (.List, .Get), а не вызов функции
	r.GET("/posts", postController.List)
	r.GET("/posts/:slug", postController.Get)

	// Эти оставляем по-старому, пока не отрефакторим CommentController и LikeController
	r.GET("/posts/:slug/comments", controllers.ListCommentsForPost(commentRepo))
	r.GET("/posts/:slug/likes", controllers.GetPostLikes(likeRepo))

	// --- Приватные роуты ---
	auth := r.Group("/posts")
	auth.Use(middleware.RequireAuth())

	// Посты: используем методы нового контроллера
	auth.POST("", postController.Create)
	auth.PUT("/:slug", postController.Update)
	auth.DELETE("/:slug", postController.Delete)

	// Лайки и Комменты: оставляем старый стиль (Factory functions)
	auth.POST("/:slug/like", controllers.LikePost(likeRepo))
	auth.DELETE("/:slug/like", controllers.UnlikePost(likeRepo))

	auth.POST("/:slug/comments", controllers.CreateComment(commentRepo))
	auth.DELETE("/comments/:id", controllers.DeleteComment(commentRepo))
}
