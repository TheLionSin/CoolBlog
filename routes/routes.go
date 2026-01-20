package routes

import (
	"github.com/gin-gonic/gin"
	"go_blog/config"
	"go_blog/internal/repositories"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	postRepo := repositories.NewPostRepository(config.DB, config.RDB)
	commentRepo := repositories.NewCommentRepository(config.DB)

	RegisterAuthRoutes(r)
	RegisterUserRoutes(r)
	RegisterPostRoutes(r, postRepo, commentRepo)

	return r
}
