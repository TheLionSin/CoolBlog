package routes

import (
	"go_blog/controllers"
	"go_blog/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterPostRoutes(r *gin.Engine) {
	r.GET("/posts", controllers.ListPosts)
	r.GET("/posts/:slug", controllers.GetPost)

	r.GET("/posts/:slug/comments", controllers.ListCommentsForPost)

	r.GET("/posts/:slug/likes", controllers.GetPostLikes)

	auth := r.Group("/posts")
	auth.Use(middleware.RequireAuth())

	auth.POST("", controllers.CreatePost)
	auth.PUT("/:slug", controllers.UpdatePost)
	auth.DELETE("/:slug", controllers.DeletePost)

	auth.POST("/:slug/like", controllers.LikePost)
	auth.DELETE("/:slug/like", controllers.UnlikePost)

	auth.POST("/:slug/comments", controllers.CreateComment)
	auth.DELETE("/comments/:id", controllers.DeleteComment)
}
