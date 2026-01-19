package controllers

import (
	"errors"
	"fmt"
	"go_blog/config"
	"go_blog/dto"
	"go_blog/helpers"
	"go_blog/internal/repositories"
	"go_blog/models"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func CreatePost(c *gin.Context) {
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

	var post models.Post

	slug, err := helpers.GenerateUniqueSlug(req.Title)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to generate slug")
		return
	}

	post = models.Post{
		Title:  req.Title,
		Text:   req.Text,
		Slug:   slug,
		UserID: uid,
	}

	if err := config.DB.Create(&post).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to create post")
		return
	}

	_ = config.RDB.Incr(c.Request.Context(), utils.PostsListVersionKey()).Err()

	utils.RespondOK(c, postToResp(post))

}

func UpdatePost(c *gin.Context) {
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

	post, ok := LoadPostOwnedBy(c, slug, uid)
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

	if err := config.DB.Model(post).Updates(updates).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to update post")
		return
	}

	if err := config.DB.First(post, post.ID).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to update post")
		return
	}

	cacheKey := "post:slug:" + slug
	_ = config.RDB.Del(c.Request.Context(), cacheKey).Err()
	_ = config.RDB.Incr(c.Request.Context(), utils.PostsListVersionKey()).Err()

	utils.RespondOK(c, postToResp(*post))

}

func DeletePost(c *gin.Context) {
	slug := c.Param("slug")

	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	post, ok := LoadPostOwnedBy(c, slug, uid)
	if !ok {
		return
	}

	if err := config.DB.Delete(post).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to delete post")
		return
	}

	cacheKey := "post:slug:" + slug
	_ = config.RDB.Del(c.Request.Context(), cacheKey).Err()
	_ = config.RDB.Incr(c.Request.Context(), utils.PostsListVersionKey()).Err()

	c.Status(http.StatusNoContent)
}

func ListPosts(c *gin.Context) {

	page, limit := utils.GetPage(c)
	q := c.Query("q")

	repo := repositories.NewPostRepository(config.DB, config.RDB)

	out, err := repo.List(c.Request.Context(), page, limit, q)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to list posts")
		return
	}

	utils.RespondOK(c, out)
}

func GetPost(c *gin.Context) {
	slug := c.Param("slug")

	postRepo := repositories.NewPostRepository(config.DB, config.RDB)

	resp, err := postRepo.GetBySlug(c.Request.Context(), slug)
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

func postToResp(p models.Post) dto.PostResponse {
	return dto.PostResponse{
		ID:        p.ID,
		Title:     p.Title,
		Text:      p.Text,
		Slug:      p.Slug,
		UserID:    p.UserID,
		IsActive:  p.IsActive,
		CreatedAt: p.CreatedAt.Format("02.01.2006 15:04"),
		UpdatedAt: p.CreatedAt.Format("02.01.2006 15:04"),
	}
}
