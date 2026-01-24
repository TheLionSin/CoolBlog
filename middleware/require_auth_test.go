package middleware_test

import (
	"go_blog/middleware"
	"go_blog/testhelpers"
	"go_blog/utils"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func setupProtectedApp() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.RequireAuth())

	r.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	return r
}

func TestRequireAuth_OK(t *testing.T) {
	app := setupProtectedApp()

	token, err := utils.GenerateAccessJWT(1, "user")
	require.NoError(t, err)

	req := testhelpers.NewAuthRequest("GET", "/protected", token)
	resp := testhelpers.DoRequest(app, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestRequireAuth_NoToken(t *testing.T) {
	app := setupProtectedApp()

	req := testhelpers.NewAuthRequest("GET", "/protected", "")
	resp := testhelpers.DoRequest(app, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	app := setupProtectedApp()

	req := testhelpers.NewAuthRequest("GET", "/protected", "not-a-token")
	resp := testhelpers.DoRequest(app, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	app := setupProtectedApp()

	req := testhelpers.NewAuthRequest("GET", "/protected", expiredToken())
	resp := testhelpers.DoRequest(app, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func expiredToken() string {
	claims := jwt.MapClaims{
		"sub":  1,
		"role": "user",
		"iat":  time.Now().Add(-2 * time.Hour).Unix(),
		"exp":  time.Now().Add(-1 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, _ := t.SignedString([]byte("asd@123#"))
	return token
}
