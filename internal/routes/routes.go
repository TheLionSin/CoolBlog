package routes

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go_blog/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRoutes(authService *services.AuthService, userService *services.UserService,
	postService *services.PostService, commentService *services.CommentService,
	likeService *services.LikeService, redisClient *redis.Client) *gin.Engine {
	r := gin.Default()

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	RegisterAuthRoutes(r, authService, redisClient)
	RegisterUserRoutes(r, userService)
	RegisterPostRoutes(r, postService, commentService, likeService)

	return r
}
