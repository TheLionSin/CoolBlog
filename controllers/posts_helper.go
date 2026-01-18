package controllers

import (
	"go_blog/config"
	"go_blog/models"
	"go_blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func LoadPostOwnedBy(c *gin.Context, slug string, uid uint) (*models.Post, bool) {
	var post models.Post
	if err := config.DB.First(&post, "slug = ?", slug).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "post not found")
		return nil, false
	}
	if post.UserID != uid {
		utils.RespondError(c, http.StatusForbidden, "user is not author")
		return nil, false
	}
	return &post, true
}
