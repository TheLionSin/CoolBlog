package controllers

import (
	"errors"
	"go_blog/internal/services"
	"go_blog/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LikeController struct {
	service *services.LikeService
}

func NewLikeController(service *services.LikeService) *LikeController {
	return &LikeController{service: service}
}

func (lc *LikeController) Like(c *gin.Context) {
	slug := c.Param("slug")
	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	err := lc.service.Like(c.Request.Context(), slug, uid)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			utils.RespondError(c, http.StatusNotFound, "post not found")
		case errors.Is(err, services.ErrAlreadyLiked):
			utils.RespondError(c, http.StatusConflict, "already liked")
		default:
			utils.RespondError(c, http.StatusInternalServerError, "failed to like")
		}
		return
	}
	utils.RespondOK(c, gin.H{"liked": true})
}

func (lc *LikeController) Unlike(c *gin.Context) {
	slug := c.Param("slug")
	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	err := lc.service.Unlike(c.Request.Context(), slug, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // Post or Like not found
			utils.RespondError(c, http.StatusNotFound, "not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to unlike")
		return
	}

	c.Status(http.StatusNoContent)
}

func (lc *LikeController) GetLikes(c *gin.Context) {
	slug := c.Param("slug")

	count, err := lc.service.GetLikesCount(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondError(c, http.StatusNotFound, "post not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to get likes")
		return
	}

	utils.RespondOK(c, gin.H{"ok": true, "likes": count})
}
