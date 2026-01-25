package controllers

import (
	"errors"
	"fmt"
	"go_blog/dto"
	"go_blog/services"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func CreatePost(postService *services.PostService) gin.HandlerFunc {
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

		resp, err := postService.Create(c.Request.Context(), uid, req)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "failed to create post")
			return
		}

		utils.RespondOK(c, resp)

	}
}

func UpdatePost(postService *services.PostService) gin.HandlerFunc {
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

		resp, err := postService.Update(c.Request.Context(), slug, uid, req)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrNoFieldsToUpdate):
				utils.RespondError(c, http.StatusBadRequest, "no fields to update")
			case errors.Is(err, services.ErrPostNotFound):
				utils.RespondError(c, http.StatusNotFound, "post not found")
			default:
				utils.RespondError(c, http.StatusInternalServerError, "failed to update post")
			}
			return
		}

		utils.RespondOK(c, resp)

	}
}

func DeletePost(postService *services.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}

		err := postService.Delete(c.Request.Context(), slug, uid)
		if err != nil {
			if errors.Is(err, services.ErrPostNotFound) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to delete post")
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func GetPost(postService *services.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {

		slug := c.Param("slug")

		resp, err := postService.Get(c.Request.Context(), slug)
		if err != nil {
			if errors.Is(err, services.ErrPostNotFound) {
				utils.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "failed to get post")
			return
		}

		utils.RespondOK(c, resp)
	}
}

func ListPosts(postService *services.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {

		page, limit := utils.GetPage(c)
		q := c.Query("q")

		out, err := postService.List(c.Request.Context(), page, limit, q)
		if err != nil {
			utils.RespondError(c, http.StatusInternalServerError, "failed to list posts")
			return
		}

		utils.RespondOK(c, out)
	}
}
