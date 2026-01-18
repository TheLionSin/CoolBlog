package controllers

import (
	"errors"
	"go_blog/config"
	"go_blog/models"
	"go_blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func LikePost(c *gin.Context) {
	slug := c.Param("slug")
	var post models.Post
	if err := config.DB.Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "post not found")
		return
	}
	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	like := models.PostLike{UserID: uid, PostID: post.ID}

	if err := config.DB.Create(&like).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			utils.RespondError(c, http.StatusConflict, "post already liked")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to like")
		return
	}

	utils.RespondOK(c, gin.H{"liked": true})

}

func UnlikePost(c *gin.Context) {
	slug := c.Param("slug")

	var post models.Post
	if err := config.DB.Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "post not found")
		return
	}

	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	var like models.PostLike
	if err := config.DB.Where("post_id = ? AND user_id = ?", post.ID, uid).Delete(&like).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to unlike")
		return
	}

	c.Status(http.StatusNoContent)

}

func GetPostLikes(c *gin.Context) {
	slug := c.Param("slug")
	var post models.Post
	if err := config.DB.Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "post not found")
		return
	}

	var count int64

	if err := config.DB.Model(&models.PostLike{}).Where("post_id = ?", post.ID).Count(&count).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to count likes")
		return
	}

	utils.RespondOK(c, gin.H{"ok": true, "likes": count})

}
