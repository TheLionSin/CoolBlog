package testhelpers

import (
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

	require.NoError(t, db.Migrator().DropTable(
		&models.PostLike{},
		&models.Comment{},
		&models.Post{},
		&models.User{},
		&models.RefreshToken{},
	))

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Post{},
		&models.Comment{},
		&models.PostLike{},
		&models.RefreshToken{},
	))

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
