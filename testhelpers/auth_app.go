package testhelpers

import (
	"go_blog/controllers"
	"go_blog/internal/repositories"
	"go_blog/services"
	"go_blog/stores"
	"testing"

	"github.com/gin-gonic/gin"
)

func SetupAuthTestApp(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := SetupTestDB(t)
	rdb := SetupTestRedis(t)

	userRepo := repositories.NewUserRepository(db)
	refreshStore := stores.NewRefreshRedisStore(rdb)
	authSvc := services.NewAuthService(userRepo, refreshStore)

	r := gin.New()
	r.POST("/auth/register", controllers.Register(authSvc))
	r.POST("/auth/login", controllers.Login(authSvc))
	r.POST("/auth/refresh", controllers.Refresh(authSvc))
	r.POST("/auth/logout", controllers.Logout(authSvc))

	return r
}
