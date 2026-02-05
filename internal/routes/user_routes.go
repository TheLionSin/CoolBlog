package routes

import (
	"go_blog/internal/controllers"
	"go_blog/internal/services"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.Engine, userService *services.UserService) {

	userController := controllers.NewUserController(userService)

	protected := r.Group("/users")
	protected.Use(middleware.RequireAuth())

	protected.GET("/me", userController.GetMe)    // Получить профиль
	protected.PUT("/me", userController.UpdateMe) // Обновить профиль
}
