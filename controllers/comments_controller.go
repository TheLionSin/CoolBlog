package controllers

import (
	"fmt"
	"go_blog/config"
	"go_blog/dto"
	"go_blog/models"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func CreateComment(c *gin.Context) {
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
	var post models.Post
	if err := config.DB.Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "post not found")
		return
	}

	var comment models.Comment

	comment = models.Comment{
		PostID: post.ID,
		UserID: uid,
		Text:   req.Text,
	}

	if err := config.DB.Create(&comment).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "create comment failed")
		return
	}

	utils.RespondCreated(c, commentToResp(comment))

}

func DeleteComment(c *gin.Context) {
	id := c.Param("id")
	var comment models.Comment
	if err := config.DB.Where("id = ?", id).First(&comment).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "comment not found")
		return
	}
	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}
	if comment.UserID != uid {
		utils.RespondError(c, http.StatusForbidden, "you are not author")
		return
	}
	if err := config.DB.Delete(&comment).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "delete comment failed")
		return
	}

	c.Status(http.StatusNoContent)
}

func ListCommentsForPost(c *gin.Context) {
	slug := c.Param("slug")
	var post models.Post
	if err := config.DB.Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "post not found")
		return
	}

	var comments []models.Comment
	if err := config.DB.Where("post_id = ?", post.ID).Find(&comments).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to list comments")
		return
	}

	resp := make([]dto.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		resp = append(resp, commentToResp(comment))
	}

	utils.RespondOK(c, gin.H{"ok": true, "comments": resp})

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
