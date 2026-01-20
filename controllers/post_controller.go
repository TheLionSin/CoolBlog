package controllers

import (
	"errors"
	"fmt"
	"go_blog/dto"
	"go_blog/internal/repositories"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func CreatePost(repo *repositories.PostRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.PostCreateRequest
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

		post, err := repo.Create(c.Request.Context(), uid, req.Title, req.Text)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "failed to create post")
			return
		}

		utils.RespondOK(c, utils.PostToResp(*post))

	}
}

func UpdatePost(repo *repositories.PostRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.PostUpdateRequest
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

		slug := c.Param("slug")

		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}

		updates := map[string]any{}

		if req.Title != nil {
			updates["title"] = *req.Title
		}
		if req.Text != nil {
			updates["text"] = *req.Text
		}

		if len(updates) == 0 {
			utils.RespondError(c, http.StatusBadRequest, "no fields to update")
			return
		}

		post, err := repo.UpdateOwnedBy(c.Request.Context(), slug, uid, updates)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to update post")
			return
		}

		utils.RespondOK(c, utils.PostToResp(*post))

	}
}

func DeletePost(repo *repositories.PostRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}

		err := repo.DeleteOwnedBy(c.Request.Context(), slug, uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to delete post")
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func ListPosts(repo *repositories.PostRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		page, limit := utils.GetPage(c)
		q := c.Query("q")

		out, err := repo.List(c.Request.Context(), page, limit, q)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "failed to list posts")
			return
		}

		utils.RespondOK(c, out)
	}
}

func GetPost(repo *repositories.PostRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		slug := c.Param("slug")

		resp, err := repo.GetBySlug(c.Request.Context(), slug)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to get post")
			return
		}

		utils.RespondOK(c, resp)
	}
}
