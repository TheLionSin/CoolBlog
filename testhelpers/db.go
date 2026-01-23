package testhelpers

import (
	"fmt"
	"github.com/joho/godotenv"
	"go_blog/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"testing"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

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
