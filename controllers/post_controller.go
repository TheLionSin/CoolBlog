package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"go_blog/config"
	"go_blog/dto"
	"go_blog/helpers"
	"go_blog/models"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"
	"time"

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
	ctx := c.Request.Context()
	page, limit := utils.GetPage(c)
	q := c.Query("q")

	var ver int64 = 1
	if config.RDB != nil {
		if v, err := config.RDB.Get(ctx, utils.PostsListVersionKey()).Int64(); err == nil {
			ver = v
		}
	}

	cacheKey := utils.PostsListsCacheKey(ver, page, limit, q)

	if config.RDB != nil {
		cached, err := config.RDB.Get(ctx, cacheKey).Result()
		if err == nil {
			var out map[string]any
			if json.Unmarshal([]byte(cached), &out) == nil {
				// чтобы page/limit всегда соответствовали текущему запросу (на всякий)
				out["page"] = page
				out["limit"] = limit
				utils.RespondOK(c, out)
				return
			}
		}
	}

	db := config.DB.Model(&models.Post{}).Where("is_active = ?", true).Order("created_at desc")

	if q != "" {
		db = db.Where("title ILIKE ?", "%"+q+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to count posts")
		return
	}

	var posts []models.Post
	if err := db.Limit(limit).Offset(utils.Offset(page, limit)).Find(&posts).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "failed to list posts")
		return
	}

	resp := make([]dto.PostResponse, 0, len(posts))
	for _, post := range posts {
		resp = append(resp, postToResp(post))
	}

	out := gin.H{
		"ok":    true,
		"page":  page,
		"limit": limit,
		"total": total,
		"posts": resp,
	}

	if config.RDB != nil {
		b, _ := json.Marshal(out)
		_ = config.RDB.Set(ctx, cacheKey, b, 30*time.Second).Err()
	}

	utils.RespondOK(c, out)
}

func GetPost(c *gin.Context) {
	slug := c.Param("slug")
	ctx := c.Request.Context()

	cacheKey := "post:slug:" + slug

	// Пытаемся взять из Redis
	if config.RDB == nil {
		cached, err := config.RDB.Get(ctx, cacheKey).Result()
		if err == nil {
			var resp dto.PostResponse
			if err := json.Unmarshal([]byte(cached), &resp); err == nil {
				utils.RespondOK(c, resp)
				return
			}
		}
	}

	var post models.Post
	if err := config.DB.Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondError(c, http.StatusNotFound, "post not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to get post")
		return
	}

	resp := postToResp(post)

	if config.RDB != nil {
		b, _ := json.Marshal(resp)
		_ = config.RDB.Set(ctx, cacheKey, b, 60*time.Second).Err()
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
