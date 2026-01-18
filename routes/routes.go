package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "welcome"})
	})

	RegisterAuthRoutes(r)
	RegisterUserRoutes(r)
	RegisterPostRoutes(r)

	return r
}
