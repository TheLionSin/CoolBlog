package routes

import (
	"go_blog/controllers"
	"go_blog/middleware"
	"go_blog/services"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.Engine, userService *services.UserService) {
	protected := r.Group("/user")
	protected.Use(middleware.RequireAuth())

	protected.GET("/me", controllers.GetCurrentUser(userService))
}
