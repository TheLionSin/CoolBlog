package controllers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go_blog/internal/dto"
	"go_blog/internal/services"
	"go_blog/internal/utils"
	"go_blog/internal/validators"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

type CommentController struct {
	service *services.CommentService
}

func NewCommentController(service *services.CommentService) *CommentController {
	return &CommentController{
		service: service,
	}
}

func (cc *CommentController) Create(c *gin.Context) {
	var req dto.CommentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid json")
		return
	}

	if err := validators.Validate.Struct(req); err != nil {
		errorsMap := make(map[string]string)
		for _, e := range err.(validator.ValidationErrors) {
			errorsMap[e.Field()] = fmt.Sprintf("validation failed: %s", e.Tag())
		}
		utils.RespondValidation(c, errorsMap)
		return
	}

	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}
	slug := c.Param("slug")

	comment, err := cc.service.Create(c.Request.Context(), slug, uid, req.Text)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			utils.RespondError(c, http.StatusNotFound, "post not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to create comment")
		return
	}

	resp := dto.CommentResponse{
		ID:        comment.ID,
		Text:      comment.Text,
		PostID:    comment.PostID,
		UserID:    comment.UserID,
		CreatedAt: comment.CreatedAt,
	}
	utils.RespondCreated(c, resp)
}

func (cc *CommentController) Delete(c *gin.Context) {
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

	err = cc.service.Delete(c.Request.Context(), uint(id), uid)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			utils.RespondError(c, http.StatusForbidden, "not your comment")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondError(c, http.StatusNotFound, "comment not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to delete comment")
		return
	}

	c.Status(http.StatusNoContent)
}

func (cc *CommentController) List(c *gin.Context) {
	slug := c.Param("slug")
	comments, err := cc.service.List(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondError(c, http.StatusNotFound, "post not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to list comments")
		return
	}

	utils.RespondOK(c, gin.H{"ok": true, "comments": comments})
}
