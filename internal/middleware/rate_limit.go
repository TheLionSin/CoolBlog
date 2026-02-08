package middleware

import (
	"fmt"
	"go_blog/internal/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit принимает клиент Redis извне (Dependency Injection).
func RateLimit(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {

	return func(c *gin.Context) {
		// 1. Fail Open: Если клиент не передан (например, Redis отключен в конфиге),
		// мы просто пропускаем запрос, не ломая сайт.
		if client == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		// Ключ: rate_limit:<IP>:<METHOD>:<PATH>
		// Пример: rl:192.168.1.1:POST:/posts
		key := fmt.Sprintf("rl:%s:%s:%s", ip, c.Request.Method, c.FullPath())
		ctx := c.Request.Context()

		// 2. Использование Pipeline (Труба)
		// Мы отправляем команды INCR и EXPIRE одним пакетом по сети.
		// Это быстрее и надежнее.
		pipe := client.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, window) // Обновляем таймер при каждом чихе (самый надежный способ для простого RL)

		// Выполняем пачку
		_, err := pipe.Exec(ctx)

		// 3. Fail Open при ошибке Redis
		if err != nil {
			// Логируем ошибку (лучше в нормальный логгер), но НЕ падаем
			fmt.Printf("RateLimit Redis error: %v\n", err)
			c.Next()
			return
		}

		// Получаем результат инкремента
		count := incr.Val()

		if count > int64(limit) {
			utils.RespondError(c, http.StatusTooManyRequests, "too many requests, slow down")
			c.Abort() // Обязательно Abort, чтобы дальше не пошло
			return
		}

		c.Next()
	}
}
