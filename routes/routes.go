package routes

import (
	"github.com/gin-gonic/gin"
	"go_blog/config"
	"go_blog/internal/repositories"
	"go_blog/services"
	"go_blog/stores"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	postRepo := repositories.NewPostRepository(config.DB, config.RDB)
	commentRepo := repositories.NewCommentRepository(config.DB)
	likeRepo := repositories.NewLikeRepository(config.DB)
	userRepo := repositories.NewUserRepository(config.DB)

	//stores
	refreshStore := stores.NewRefreshRedisStore(config.RDB)

	//services
	authService := services.NewAuthService(userRepo, refreshStore)
	userService := services.NewUserService(userRepo)

	RegisterAuthRoutes(r, authService)
	RegisterUserRoutes(r, userService)
	RegisterPostRoutes(r, postRepo, commentRepo, likeRepo)

	return r
}
