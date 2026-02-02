package controllers

import (
	"context"
	"errors"
	"fmt"
	"go_blog/internal/dto"
	services2 "go_blog/internal/services"
	utils2 "go_blog/internal/utils"
	"go_blog/validators"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func CreatePost(postService *services2.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.PostCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils2.RespondError(c, http.StatusBadRequest, "invalid json")
			return
		}

		if err := validators.Validate.Struct(req); err != nil {
			errorsMap := make(map[string]string)
			for _, e := range err.(validator.ValidationErrors) {
				errorsMap[e.Field()] = fmt.Sprintf("не проходит '%s'", e.Tag())
			}
			utils2.RespondValidation(c, errorsMap)
			return
		}

		uid, ok := utils2.MustUserID(c)
		if !ok {
			return
		}

		post, err := postService.Create(context.Background(), uid, req.Title, req.Text)
		if err != nil {
			utils2.RespondError(c, http.StatusInternalServerError, "failed to create post")
			return
		}

		utils2.RespondOK(c, utils2.PostToResp(*post))

	}
}

func UpdatePost(postService *services2.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.PostUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils2.RespondError(c, http.StatusBadRequest, "invalid json")
			return
		}
		if err := validators.Validate.Struct(req); err != nil {
			errorsMap := make(map[string]string)
			for _, e := range err.(validator.ValidationErrors) {
				errorsMap[e.Field()] = fmt.Sprintf("не проходит '%s'", e.Tag())
			}
			utils2.RespondValidation(c, errorsMap)
			return
		}

		slug := c.Param("slug")

		uid, ok := utils2.MustUserID(c)
		if !ok {
			return
		}

		post, err := postService.Update(context.Background(), slug, uid, req.Title, req.Text)
		if err != nil {
			switch {
			case errors.Is(err, services2.ErrNoFieldsToUpdate):
				utils2.RespondError(c, http.StatusBadRequest, "no fields to update")
			case errors.Is(err, services2.ErrPostNotFound):
				utils2.RespondError(c, http.StatusNotFound, "post not found")
			default:
				utils2.RespondError(c, http.StatusInternalServerError, "failed to update post")
			}
			return
		}

		utils2.RespondOK(c, utils2.PostToResp(*post))

	}
}

func DeletePost(postService *services2.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		uid, ok := utils2.MustUserID(c)
		if !ok {
			return
		}

		err := postService.Delete(c.Request.Context(), slug, uid)
		if err != nil {
			if errors.Is(err, services2.ErrPostNotFound) {
				utils2.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils2.RespondError(c, http.StatusInternalServerError, "failed to delete post")
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func GetPost(postService *services2.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {

		slug := c.Param("slug")

		resp, err := postService.Get(c.Request.Context(), slug)
		if err != nil {
			if errors.Is(err, services2.ErrPostNotFound) {
				utils2.RespondError(c, http.StatusNotFound, "post not found")
				return
			}
			utils2.RespondError(c, http.StatusInternalServerError, "failed to get post")
			return
		}

		utils2.RespondOK(c, resp)
	}
}

func ListPosts(postService *services2.PostService) gin.HandlerFunc {
	return func(c *gin.Context) {

		page, limit := utils2.GetPage(c)
		q := c.Query("q")

		posts, total, err := postService.List(c.Request.Context(), page, limit, q)
		if err != nil {
			utils2.RespondError(c, http.StatusInternalServerError, "failed to list posts")
			return
		}

		respPosts := make([]dto.PostResponse, 0, len(posts))
		for i := range posts {
			respPosts = append(respPosts, utils2.PostToResp(posts[i]))
		}

		out := dto.PostListResponse{
			Ok:    true,
			Page:  page,
			Limit: limit,
			Total: total,
			Posts: respPosts,
		}

		utils2.RespondOK(c, out)

	}
}
