package routes

import (
	"go_blog/config"
	"go_blog/internal/adapters/eventbus/inmem"
	"go_blog/internal/repositories"
	"go_blog/services"
	"go_blog/stores"

	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	postRepo := repositories.NewPostRepository(config.DB, config.RDB)
	commentRepo := repositories.NewCommentRepository(config.DB)
	likeRepo := repositories.NewLikeRepository(config.DB)
	userRepo := repositories.NewUserRepository(config.DB)
	bus := inmem.New()

	//stores
	refreshStore := stores.NewRefreshRedisStore(config.RDB)

	//services
	authService := services.NewAuthService(userRepo, refreshStore)
	userService := services.NewUserService(userRepo)
	postService := services.NewPostService(postRepo, bus)

	RegisterAuthRoutes(r, authService)
	RegisterUserRoutes(r, userService)
	RegisterPostRoutes(r, postService, commentRepo, likeRepo)

	return r
}
