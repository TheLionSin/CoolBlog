package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go_blog/dto"
	"go_blog/helpers"
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

func postBySlugKey(slug string) string {
	return "post:slug:" + slug
}

func postsListKey(ver int64, page, limit int, q string) string {
	qNorm := strings.TrimSpace(strings.ToLower(q))
	sum := sha256.Sum256([]byte(qNorm))
	qh := hex.EncodeToString(sum[:8]) // короткий хэш, чтобы ключи не раздувались
	return fmt.Sprintf("posts:list:v%d:p%d:l%d:q%s", ver, page, limit, qh)
}

func (r *PostRepository) listVersion(ctx context.Context) int64 {
	if r.rdb == nil {
		return 1
	}
	v, err := r.rdb.Get(ctx, utils.PostsListVersionKey()).Int64()
	if err != nil {
		return 1
	}
	return v
}

func (r *PostRepository) bumpListVersion(ctx context.Context) {
	if r.rdb == nil {
		return
	}
	_ = r.rdb.Incr(ctx, utils.PostsListVersionKey()).Err()
}

func (r *PostRepository) GetBySlug(ctx context.Context, slug string) (dto.PostResponse, error) {
	cacheKey := postBySlugKey(slug)

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

	resp := utils.PostToResp(post)

	if r.rdb != nil {
		b, _ := json.Marshal(resp)
		_ = r.rdb.Set(ctx, cacheKey, b, time.Minute).Err()
	}

	return resp, nil

}

func (r *PostRepository) List(ctx context.Context, page, limit int, q string) (dto.PostListResponse, error) {

	ver := r.listVersion(ctx)

	cacheKey := postsListKey(ver, page, limit, q)

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

	db := r.db.WithContext(ctx).Model(&models.Post{}).Where("is_active = ?", true).Order("created_at desc")

	qNorm := strings.TrimSpace(q)
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
		respPosts = append(respPosts, utils.PostToResp(p))
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

func (r *PostRepository) Create(ctx context.Context, uid uint, title, text string) (*models.Post, error) {
	slug, err := helpers.GenerateUniqueSlug(title)
	if err != nil {
		return nil, err
	}

	post := &models.Post{
		Title:  title,
		Text:   text,
		Slug:   slug,
		UserID: uid,
	}

	if err := r.db.WithContext(ctx).Create(post).Error; err != nil {
		return nil, err
	}

	r.bumpListVersion(ctx)

	return post, nil
}

func (r *PostRepository) UpdateOwnedBy(ctx context.Context, slug string, uid uint, updates map[string]any) (*models.Post, error) {
	var post models.Post
	if err := r.db.WithContext(ctx).Where("slug = ? AND user_id = ? AND is_active = ?", slug, uid, true).First(&post).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).Model(&post).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).First(&post, post.ID).Error; err != nil {
		return nil, err
	}

	if r.rdb != nil {
		_ = r.rdb.Del(ctx, postBySlugKey(slug)).Err()
	}
	r.bumpListVersion(ctx)

	return &post, nil
}

func (r *PostRepository) DeleteOwnedBy(ctx context.Context, slug string, uid uint) error {
	var post models.Post
	if err := r.db.WithContext(ctx).Where("slug = ? AND user_id = ? AND is_active = ?", slug, uid, true).First(&post).Error; err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Delete(&post).Error; err != nil {
		return err
	}

	if r.rdb != nil {
		_ = r.rdb.Del(ctx, postBySlugKey(slug)).Err()
	}
	r.bumpListVersion(ctx)

	return nil
}
