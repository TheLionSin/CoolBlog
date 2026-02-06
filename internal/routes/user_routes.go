package routes

import (
	"github.com/gin-gonic/gin"
	"go_blog/internal/controllers"
	"go_blog/internal/middleware"
	"go_blog/internal/services"
)

func RegisterUserRoutes(r *gin.Engine, userService *services.UserService) {

	userController := controllers.NewUserController(userService)

	protected := r.Group("/users")
	protected.Use(middleware.RequireAuth())

	protected.GET("/me", userController.GetMe)    // Получить профиль
	protected.PUT("/me", userController.UpdateMe) // Обновить профиль
}
