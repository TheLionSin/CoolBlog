package config

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	ctx := context.Background()
	if err := RDB.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis %v", err)
	}

	log.Println("Redis connected")
}
