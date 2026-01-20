package controllers

import (
	"errors"
	"fmt"
	"go_blog/dto"
	"go_blog/internal/repositories"
	"go_blog/models"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func CreateComment(repo *repositories.CommentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req dto.CommentCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "invalid json")
			return
		}

		if err := validators.Validate.Struct(req); err != nil {
			errorsMap := make(map[string]string)
			for _, e := range err.(validator.ValidationErrors) {
				errorsMap[e.Field()] = fmt.Sprintf("не проходит '%s'", e.Tag())
			}
			utils.RespondValidation(c, errorsMap)
			return
		}

		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}

		slug := c.Param("slug")

		comment, err := repo.Create(c.Request.Context(), slug, uid, req.Text)
		if err != nil {
			if repositories.IsNotFound(err) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "create comment failed")
			return
		}

		utils.RespondCreated(c, commentToResp(*comment))

	}
}

func DeleteComment(repo *repositories.CommentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			utils.RespondError(c, http.StatusBadRequest, "invalid id")
			return
		}

		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}

		err = repo.DeleteOwnedBy(c.Request.Context(), uint(id), uid)
		if err != nil {
			if errors.Is(err, repositories.ErrForbidden) {
				utils.RespondError(c, http.StatusForbidden, "you are not author")
				return
			}
			if repositories.IsNotFound(err) {
				utils.RespondError(c, http.StatusNotFound, "comment not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "delete comment failed")
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func ListCommentsForPost(repo *repositories.CommentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		comments, err := repo.ListByPostSlug(c.Request.Context(), slug)
		if err != nil {
			if repositories.IsNotFound(err) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to list comments")
			return
		}

		resp := make([]dto.CommentResponse, 0, len(comments))
		for _, comment := range comments {
			resp = append(resp, commentToResp(comment))
		}

		utils.RespondOK(c, gin.H{"ok": true, "comments": resp})
	}
}

func commentToResp(c models.Comment) dto.CommentResponse {
	return dto.CommentResponse{
		ID:        c.ID,
		Text:      c.Text,
		PostID:    c.PostID,
		UserID:    c.UserID,
		CreatedAt: c.CreatedAt,
	}
}
