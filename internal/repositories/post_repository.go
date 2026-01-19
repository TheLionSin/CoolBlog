package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go_blog/dto"
	"go_blog/models"
	"go_blog/utils"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PostRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewPostRepository(db *gorm.DB, rdb *redis.Client) *PostRepository {
	return &PostRepository{db: db, rdb: rdb}
}

func (r *PostRepository) GetBySlug(ctx context.Context, slug string) (dto.PostResponse, error) {
	cacheKey := "post:slug:" + slug

	if r.rdb != nil {
		if cached, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var resp dto.PostResponse
			if json.Unmarshal([]byte(cached), &resp) == nil {
				return resp, nil
			}
		}
	}

	var post models.Post
	if err := r.db.Where("slug = ? AND is_active = ?", slug, true).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PostResponse{}, err
		}
		return dto.PostResponse{}, err
	}

	resp := postToResp(post)

	if r.rdb != nil {
		_ = r.rdb.Set(ctx, cacheKey, utils.MustJSON(resp), time.Minute)
	}

	return resp, nil

}

func (r *PostRepository) List(ctx context.Context, page, limit int, q string) (dto.PostListResponse, error) {

	qNorm := strings.TrimSpace(strings.ToLower(q))

	sum := sha256.Sum256([]byte(qNorm))
	qh := hex.EncodeToString(sum[:8])
	cacheKey := fmt.Sprintf("posts:list:p%d:l%d:q%s", page, limit, qh)

	if r.rdb != nil {
		if cached, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var out dto.PostListResponse
			if json.Unmarshal([]byte(cached), &out) == nil {
				out.Page = page
				out.Limit = limit
				return out, nil
			}
		}
	}

	db := r.db.Model(&models.Post{}).Where("is_active = ?", true).Order("created_at desc")

	if qNorm != "" {
		db = db.Where("title ILIKE ?", "%"+qNorm+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return dto.PostListResponse{}, err
	}

	var posts []models.Post
	offset := utils.Offset(page, limit)
	if err := db.Limit(limit).Offset(offset).Find(&posts).Error; err != nil {
		return dto.PostListResponse{}, err
	}

	respPosts := make([]dto.PostResponse, 0, len(posts))
	for _, p := range posts {
		respPosts = append(respPosts, postToResp(p))
	}

	out := dto.PostListResponse{
		Ok:    true,
		Page:  page,
		Limit: limit,
		Total: total,
		Posts: respPosts,
	}

	if r.rdb != nil {
		b, _ := json.Marshal(out)
		_ = r.rdb.Set(ctx, cacheKey, b, 30*time.Second).Err()
	}

	return out, nil

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
