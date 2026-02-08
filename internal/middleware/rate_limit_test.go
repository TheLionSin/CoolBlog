package middleware_test

import (
	"fmt"
	"go_blog/config"
	"go_blog/internal/middleware"
	testhelpers2 "go_blog/internal/testhelpers"
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
	rdb := testhelpers2.SetupTestRedis(t)

	redisClient, err := config.InitRedis()
	if err != nil {
		fmt.Printf("Failed to connect to Redis: %v", err)
	}

	prev := redisClient
	redisClient = rdb
	t.Cleanup(func() { redisClient = prev })

	require.NoError(t, rdb.FlushDB(t.Context()).Err())

	app := setupRateLimitApp()

	req := testhelpers2.NewAuthRequest("GET", "/ping", "")
	req.RemoteAddr = "127.0.0.1:12345"

	resp := testhelpers2.DoRequest(app, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = testhelpers2.DoRequest(app, req)
	require.Equal(t, http.StatusOK, resp.Code)

	resp = testhelpers2.DoRequest(app, req)
	require.Equal(t, http.StatusTooManyRequests, resp.Code)

}

func TestRateLimit_NoRedis_Allows(t *testing.T) {
	redisClient, err := config.InitRedis()
	if err != nil {
		fmt.Printf("Failed to connect to Redis: %v", err)
	}
	prev := redisClient
	redisClient = nil
	t.Cleanup(func() { redisClient = prev })

	app := setupRateLimitApp()

	req := testhelpers2.NewAuthRequest("GET", "/ping", "")
	req.RemoteAddr = "127.0.0.1:12345"

	for i := 0; i < 10; i++ {
		resp := testhelpers2.DoRequest(app, req)
		require.Equal(t, http.StatusOK, resp.Code)
	}

}

func TestRateLimit_ResetsAfterWindow(t *testing.T) {
	rdb := testhelpers2.SetupTestRedis(t)

	redisClient, err := config.InitRedis()
	if err != nil {
		fmt.Printf("Failed to connect to Redis: %v", err)
	}

	prev := redisClient
	redisClient = rdb
	t.Cleanup(func() { redisClient = prev })

	require.NoError(t, rdb.FlushDB(t.Context()).Err())

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/ping", middleware.RateLimit(1, 1*time.Second), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := testhelpers2.NewAuthRequest("GET", "/ping", "")
	req.RemoteAddr = "127.0.0.1:12345"

	// 1-й — OK
	resp := testhelpers2.DoRequest(r, req)
	require.Equal(t, http.StatusOK, resp.Code)

	// 2-й — лимит
	resp = testhelpers2.DoRequest(r, req)
	require.Equal(t, http.StatusTooManyRequests, resp.Code)

	// ждём, пока TTL истечёт
	time.Sleep(2 * time.Second)

	// снова OK
	resp = testhelpers2.DoRequest(r, req)
	require.Equal(t, http.StatusOK, resp.Code)
}
