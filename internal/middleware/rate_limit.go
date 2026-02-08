package middleware

import (
	"fmt"
	"go_blog/config"
	"go_blog/internal/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimit(limit int, window time.Duration) gin.HandlerFunc {

	redisClient, err := config.InitRedis()
	if err != nil {
		fmt.Printf("Failed to connect to Redis: %v", err)
	}

	return func(c *gin.Context) {
		if redisClient == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		ip := c.ClientIP()
		key := "rl:" + ip + ":" + c.FullPath()

		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			_ = redisClient.Expire(ctx, key, window).Err()
		}

		if count > int64(limit) {
			utils.RespondError(c, http.StatusTooManyRequests, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}
