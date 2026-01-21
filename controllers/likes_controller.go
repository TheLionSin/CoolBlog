package controllers

import (
	"errors"
	"go_blog/internal/repositories"
	"go_blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func LikePost(repo *repositories.LikeRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}

		err := repo.Like(c.Request.Context(), slug, uid)
		if err != nil {
			if repositories.IsNotFound(err) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			if errors.Is(err, repositories.ErrAlreadyLiked) {
				utils.RespondError(c, http.StatusConflict, "post already liked")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to like")
			return
		}

		utils.RespondOK(c, gin.H{"liked": true})

	}
}

func UnlikePost(repo *repositories.LikeRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		slug := c.Param("slug")

		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}

		err := repo.Unlike(c.Request.Context(), slug, uid)
		if err != nil {
			if repositories.IsNotFound(err) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to unlike")
			return
		}

		c.Status(http.StatusNoContent)

	}
}

func GetPostLikes(repo *repositories.LikeRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		count, err := repo.CountByPostSlug(c.Request.Context(), slug)
		if err != nil {
			if repositories.IsNotFound(err) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to count likes")
			return
		}

		utils.RespondOK(c, gin.H{"ok": true, "likes": count})

	}
}
