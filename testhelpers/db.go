package testhelpers

import (
	"fmt"
	"go_blog/models"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	_ = godotenv.Load(".env.test")
	_ = godotenv.Load("../.env.test")
	_ = godotenv.Load("../../.env.test")

	err := godotenv.Load(".env.test")

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Fatal("TEST_DB_DSN is empty (set env or .env.test)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Post{}, &models.Comment{}, &models.PostLike{}, &models.RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	require.NoError(t, db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error)
	require.NoError(t, db.Exec("TRUNCATE TABLE posts RESTART IDENTITY CASCADE").Error)
	require.NoError(t, db.Exec("TRUNCATE TABLE comments RESTART IDENTITY CASCADE").Error)
	require.NoError(t, db.Exec("TRUNCATE TABLE post_likes RESTART IDENTITY CASCADE").Error)

	return db
}

func BeginTx(t *testing.T, db *gorm.DB) *gorm.DB {
	t.Helper()

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin tx: %v", tx.Error)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return tx
}

func RequireNotZero(t *testing.T, v any, msg string) {
	t.Helper()
	if fmt.Sprint(v) == "0" || fmt.Sprint(v) == "<nil>" {
		t.Fatalf("expected not zero: %s", msg)
	}
}
