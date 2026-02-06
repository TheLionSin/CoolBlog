package controllers_test

import (
	"go_blog/internal/controllers"
	"go_blog/internal/repositories"
	"go_blog/internal/services"
	"go_blog/internal/stores"
	testhelpers2 "go_blog/internal/testhelpers"
	"testing"

	"github.com/gin-gonic/gin"
)

func SetupAuthTestApp(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testhelpers2.SetupTestDB(t)
	rdb := testhelpers2.SetupTestRedis(t)

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
