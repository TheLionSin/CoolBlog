package routes

import (
	controllers2 "go_blog/internal/controllers"
	"go_blog/internal/repositories"
	"go_blog/internal/services"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPostRoutes(r *gin.Engine,
	postService *services.PostService,
	commentRepo *repositories.CommentRepository,
	likeRepo *repositories.LikeRepository) {
	r.GET("/posts", controllers2.ListPosts(postService))
	r.GET("/posts/:slug", controllers2.GetPost(postService))

	r.GET("/posts/:slug/comments", controllers2.ListCommentsForPost(commentRepo))

	r.GET("/posts/:slug/likes", controllers2.GetPostLikes(likeRepo))

	auth := r.Group("/posts")
	auth.Use(middleware.RequireAuth())

	auth.POST("", controllers2.CreatePost(postService))
	auth.PUT("/:slug", controllers2.UpdatePost(postService))
	auth.DELETE("/:slug", controllers2.DeletePost(postService))

	auth.POST("/:slug/like", controllers2.LikePost(likeRepo))
	auth.DELETE("/:slug/like", controllers2.UnlikePost(likeRepo))

	auth.POST("/:slug/comments", controllers2.CreateComment(commentRepo))
	auth.DELETE("/comments/:id", controllers2.DeleteComment(commentRepo))
}
