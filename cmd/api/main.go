package main

import (
	"context"
	"errors"
	"go_blog/config"
	models2 "go_blog/internal/models"
	"go_blog/internal/repositories"
	"go_blog/internal/services"
	"go_blog/routes"
	"go_blog/stores"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	// 1. Инициализация конфига

	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	rdb := config.InitRedis()

	config.DB.AutoMigrate(&models2.User{}, &models2.Post{}, &models2.RefreshToken{}, &models2.PostLike{}, &models2.Comment{}, &models2.AuditLog{}, &models2.OutboxEvent{})

	userRepo := repositories.NewUserRepository(db)
	postRepo := repositories.NewPostRepository(db, rdb)
	commentRepo := repositories.NewCommentRepository(db)
	likeRepo := repositories.NewLikeRepository(db)
	outboxRepo := repositories.NewOutboxRepository(db)

	refreshStore := stores.NewRefreshRedisStore(rdb)

	authService := services.NewAuthService(userRepo, refreshStore)
	userService := services.NewUserService(userRepo)
	postService := services.NewPostService(db, postRepo, outboxRepo)

	r := routes.SetupRoutes(authService, userService, postService, commentRepo, likeRepo)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")

}
