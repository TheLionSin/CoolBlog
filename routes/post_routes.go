package routes

import (
	"go_blog/controllers"
	"go_blog/internal/repositories"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPostRoutes(r *gin.Engine,
	postRepo *repositories.PostRepository,
	commentRepo *repositories.CommentRepository) {
	r.GET("/posts", controllers.ListPosts(postRepo))
	r.GET("/posts/:slug", controllers.GetPost(postRepo))

	r.GET("/posts/:slug/comments", controllers.ListCommentsForPost(commentRepo))

	r.GET("/posts/:slug/likes", controllers.GetPostLikes)

	auth := r.Group("/posts")
	auth.Use(middleware.RequireAuth())

	auth.POST("", controllers.CreatePost(postRepo))
	auth.PUT("/:slug", controllers.UpdatePost(postRepo))
	auth.DELETE("/:slug", controllers.DeletePost(postRepo))

	auth.POST("/:slug/like", controllers.LikePost)
	auth.DELETE("/:slug/like", controllers.UnlikePost)

	auth.POST("/:slug/comments", controllers.CreateComment(commentRepo))
	auth.DELETE("/comments/:id", controllers.DeleteComment(commentRepo))
}
