package main

import (
	"go_blog/config"
	"go_blog/models"
	"go_blog/routes"
)

func main() {

	config.ConnectDB()
	config.InitRedis()
	config.DB.AutoMigrate(&models.User{}, &models.Post{}, &models.RefreshToken{}, &models.PostLike{}, &models.Comment{}, &models.AuditLog{})
	
	r := routes.SetupRoutes()

	r.Run(":8080")

}
