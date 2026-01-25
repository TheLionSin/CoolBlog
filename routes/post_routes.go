package routes

import (
	"go_blog/controllers"
	"go_blog/internal/repositories"
	"go_blog/middleware"
	"go_blog/services"

	"github.com/gin-gonic/gin"
)

func RegisterPostRoutes(r *gin.Engine,
	postService *services.PostService,
	commentRepo *repositories.CommentRepository,
	likeRepo *repositories.LikeRepository) {
	r.GET("/posts", controllers.ListPosts(postService))
	r.GET("/posts/:slug", controllers.GetPost(postService))

	r.GET("/posts/:slug/comments", controllers.ListCommentsForPost(commentRepo))

	r.GET("/posts/:slug/likes", controllers.GetPostLikes(likeRepo))

	auth := r.Group("/posts")
	auth.Use(middleware.RequireAuth())

	auth.POST("", controllers.CreatePost(postService))
	auth.PUT("/:slug", controllers.UpdatePost(postService))
	auth.DELETE("/:slug", controllers.DeletePost(postService))

	auth.POST("/:slug/like", controllers.LikePost(likeRepo))
	auth.DELETE("/:slug/like", controllers.UnlikePost(likeRepo))

	auth.POST("/:slug/comments", controllers.CreateComment(commentRepo))
	auth.DELETE("/comments/:id", controllers.DeleteComment(commentRepo))
}
