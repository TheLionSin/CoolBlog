package routes

import (
	services "go_blog/internal/services"

	"go_blog/internal/repositories"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(authService *services.AuthService, userService *services.UserService, postService *services.PostService, commentRepo *repositories.CommentRepository, likeRepo *repositories.LikeRepository) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	RegisterAuthRoutes(r, authService)
	RegisterUserRoutes(r, userService)
	RegisterPostRoutes(r, postService, commentRepo, likeRepo)

	return r
}
