package testhelpers

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func SetupTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	_ = godotenv.Load(".env.test")
	_ = godotenv.Load("../.env.test")
	_ = godotenv.Load("../../.env.test")

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Fatal("TEST_REDIS_ADDR is empty")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   1,
	})

	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("failed to flush redis: %v", err)
	}

	t.Cleanup(func() {
		_ = rdb.FlushDB(context.Background()).Err()
		_ = rdb.Close()
	})
	return rdb
}
