package routes

import (
	"go_blog/internal/controllers"
	"go_blog/internal/services"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.Engine, userService *services.UserService) {
	protected := r.Group("/user")
	protected.Use(middleware.RequireAuth())

	protected.GET("/me", controllers.GetCurrentUser(userService))
}
