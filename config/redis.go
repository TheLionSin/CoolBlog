package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func InitRedis() (*redis.Client, error) {

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0, //default
	})

	// Ping с таймаутом (Good Practice!)
	// Если Редис висит, мы не хотим ждать вечно. Ждем 5 секунд и падаем, если нет ответа.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		// Возвращаем ошибку наверх, не убиваем приложение тут (log.Fatal лучше в main)
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Redis connected successfully")

	return client, nil
}
