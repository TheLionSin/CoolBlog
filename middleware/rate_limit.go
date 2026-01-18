package middleware

import (
	"go_blog/config"
	"go_blog/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.RDB == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		ip := c.ClientIP()
		key := "rl:" + ip + ":" + c.FullPath()

		count, err := config.RDB.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			_ = config.RDB.Expire(ctx, key, window).Err()
		}

		if count > int64(limit) {
			utils.RespondError(c, http.StatusTooManyRequests, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}
