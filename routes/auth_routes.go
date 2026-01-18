package routes

import (
	"go_blog/controllers"
	"go_blog/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.Engine) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", middleware.RateLimit(5, time.Minute), controllers.Login)
		auth.POST("/refresh", controllers.Refresh)
		auth.POST("/logout", controllers.Logout)
	}
}
