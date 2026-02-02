package routes

import (
	"go_blog/internal/controllers"
	"go_blog/internal/services"
	"go_blog/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.Engine, auth *services.AuthService) {
	group := r.Group("/auth")
	{
		group.POST("/register", controllers.Register(auth))
		group.POST("/login", middleware.RateLimit(5, time.Minute), controllers.Login(auth))
		group.POST("/refresh", controllers.Refresh(auth))
		group.POST("/logout", controllers.Logout(auth))
	}
}
