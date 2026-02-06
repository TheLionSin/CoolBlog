package controllers

import (
	"context"
	"errors"
	"fmt"
	"go_blog/internal/dto"
	"go_blog/internal/services"
	"go_blog/internal/utils"
	"go_blog/internal/validators"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type PostController struct {
	service *services.PostService
}

func NewPostController(service *services.PostService) *PostController {
	return &PostController{service: service}
}

func (pc *PostController) Create(c *gin.Context) {
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

	post, err := pc.service.Create(c.Request.Context(), uid, req.Title, req.Text)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to create post")
		return
	}

	utils.RespondOK(c, utils.PostToResp(*post))

}

func (pc *PostController) Update(c *gin.Context) {

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

	post, err := pc.service.Update(context.Background(), slug, uid, req.Title, req.Text)
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

	utils.RespondOK(c, utils.PostToResp(*post))

}

func (pc *PostController) Delete(c *gin.Context) {
	slug := c.Param("slug")
	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	err := pc.service.Delete(c.Request.Context(), slug, uid)
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

func (pc *PostController) Get(c *gin.Context) {
	slug := c.Param("slug")

	resp, err := pc.service.Get(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, services.ErrPostNotFound) {
			utils.RespondError(c, http.StatusNotFound, "post not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to get post")
		return
	}

	utils.RespondOK(c, utils.PostToResp(*resp))
}

func (pc *PostController) List(c *gin.Context) {

	page, limit := utils.GetPage(c)
	q := c.Query("q")

	posts, total, err := pc.service.List(c.Request.Context(), page, limit, q)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to list posts")
		return
	}

	respPosts := make([]dto.PostResponse, 0, len(posts))
	for i := range posts {
		respPosts = append(respPosts, utils.PostToResp(posts[i]))
	}

	out := dto.PostListResponse{
		Ok:    true,
		Page:  page,
		Limit: limit,
		Total: total,
		Posts: respPosts,
	}

	utils.RespondOK(c, out)

}
