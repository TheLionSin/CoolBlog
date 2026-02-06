package routes

import (
	"go_blog/internal/controllers"
	"go_blog/internal/middleware"
	"go_blog/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.Engine, authService *services.AuthService) {
	authController := controllers.NewAuthController(authService)

	group := r.Group("/auth")
	{
		group.POST("/register", authController.Register)
		group.POST("/login", middleware.RateLimit(5, time.Minute), authController.Login)
		group.POST("/refresh", authController.Refresh)
		group.POST("/logout", authController.Logout)
	}
}
