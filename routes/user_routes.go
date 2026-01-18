package routes

import (
	"go_blog/controllers"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.Engine) {
	protected := r.Group("user/")
	protected.Use(middleware.RequireAuth())

	protected.GET("me", controllers.GetCurrentUser)
}
