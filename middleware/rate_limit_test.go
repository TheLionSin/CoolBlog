package middleware_test

import (
	"go_blog/config"
	"go_blog/middleware"
	"go_blog/testhelpers"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupRateLimitApp() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	// важно: именно роут, чтобы FullPath() был "/ping"
	r.GET("/ping", middleware.RateLimit(2, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return r
}

func TestRateLimit_AllowsThen429(t *testing.T) {
	rdb := testhelpers.SetupTestRedis(t)

	prev := config.RDB
	config.RDB = rdb
	t.Cleanup(func() { config.RDB = prev })

	require.NoError(t, rdb.FlushDB(t.Context()).Err())

	app := setupRateLimitApp()

	req := testhelpers.NewAuthRequest("GET", "/ping", "")
	req.RemoteAddr = "127.0.0.1:12345"

	resp := testhelpers.DoRequest(app, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = testhelpers.DoRequest(app, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = testhelpers.DoRequest(app, req)
	require.Equal(t, http.StatusTooManyRequests, resp.Code)

}

func TestRateLimit_NoRedis_Allows(t *testing.T) {
	prev := config.RDB
	config.RDB = nil
	t.Cleanup(func() { config.RDB = prev })

	app := setupRateLimitApp()

	req := testhelpers.NewAuthRequest("GET", "/ping", "")
	req.RemoteAddr = "127.0.0.1:12345"

	for i := 0; i < 10; i++ {
		resp := testhelpers.DoRequest(app, req)
		require.Equal(t, http.StatusOK, resp.Code)
	}

}

func TestRateLimit_ResetsAfterWindow(t *testing.T) {
	rdb := testhelpers.SetupTestRedis(t)

	prev := config.RDB
	config.RDB = rdb
	t.Cleanup(func() { config.RDB = prev })

	require.NoError(t, rdb.FlushDB(t.Context()).Err())

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/ping", middleware.RateLimit(1, 1*time.Second), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := testhelpers.NewAuthRequest("GET", "/ping", "")
	req.RemoteAddr = "127.0.0.1:12345"

	// 1-й — OK
	resp := testhelpers.DoRequest(r, req)
	require.Equal(t, http.StatusOK, resp.Code)

	// 2-й — лимит
	resp = testhelpers.DoRequest(r, req)
	require.Equal(t, http.StatusTooManyRequests, resp.Code)

	// ждём, пока TTL истечёт
	time.Sleep(2 * time.Second)

	// снова OK
	resp = testhelpers.DoRequest(r, req)
	require.Equal(t, http.StatusOK, resp.Code)
}
