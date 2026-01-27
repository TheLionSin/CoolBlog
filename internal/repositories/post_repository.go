package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func generateUniqueSlugWithDB(
	ctx context.Context,
	db *gorm.DB,
	title string,
) (string, error) {

	base := utils.Slugify(title)
	slug := base

	for i := 1; ; i++ {
		var count int64
		err := db.WithContext(ctx).
			Model(&models.Post{}).
			Where("slug = ?", slug).
			Count(&count).Error
		if err != nil {
			return "", err
		}

		if count == 0 {
			return slug, nil
		}

		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func (r *PostRepository) generateUniqueSlug(
	ctx context.Context,
	title string,
) (string, error) {
	return generateUniqueSlugWithDB(ctx, r.db, title)
}

func (r *PostRepository) listVersion(ctx context.Context) int64 {
	if r.rdb == nil {
		return 1
	}

	key := utils.PostsListVersionKey()

	v, err := r.rdb.Get(ctx, key).Int64()
	if err == nil {
		return v
	}

	_ = r.rdb.Set(ctx, key, 1, 0).Err()
	return 1
}

func (r *PostRepository) bumpListVersion(ctx context.Context) {
	if r.rdb == nil {
		return
	}
	_ = r.rdb.Incr(ctx, utils.PostsListVersionKey()).Err()
}

func (r *PostRepository) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	cacheKey := postBySlugKey(slug)

	if r.rdb != nil {
		if cached, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cp cachedPost
			if json.Unmarshal([]byte(cached), &cp) == nil {
				return cp.toModel(), nil
			}
		}
	}

	var post models.Post
	if err := r.db.WithContext(ctx).
		Select("id", "created_at", "updated_at", "title", "text", "slug", "user_id", "is_active").
		Where("slug = ? AND is_active = ?", slug, true).
		First(&post).Error; err != nil {
		return nil, err
	}

	if r.rdb != nil {
		if b, err := json.Marshal(toCachedPost(post)); err == nil {
			_ = r.rdb.Set(ctx, cacheKey, b, time.Minute).Err()
		}
	}

	return &post, nil
}

func (r *PostRepository) List(ctx context.Context, page, limit int, q string) ([]models.Post, int64, error) {
	ver := r.listVersion(ctx)
	cacheKey := postsListKey(ver, page, limit, q)

	if r.rdb != nil {
		if cached, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cl cachedPostList
			if json.Unmarshal([]byte(cached), &cl) == nil {
				posts := make([]models.Post, 0, len(cl.Posts))
				for i := range cl.Posts {
					posts = append(posts, *cl.Posts[i].toModel())
				}
				return posts, cl.Total, nil
			}
		}
	}

	db := r.db.WithContext(ctx).
		Model(&models.Post{}).
		Where("is_active = ?", true)

	qNorm := strings.TrimSpace(q)
	if qNorm != "" {
		db = db.Where("title ILIKE ?", "%"+qNorm+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var posts []models.Post
	offset := utils.Offset(page, limit)
	if err := db.
		Select("id", "created_at", "updated_at", "title", "text", "slug", "user_id", "is_active").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	if r.rdb != nil {
		cposts := make([]cachedPost, 0, len(posts))
		for i := range posts {
			cposts = append(cposts, toCachedPost(posts[i]))
		}
		if b, err := json.Marshal(cachedPostList{Total: total, Posts: cposts}); err == nil {
			_ = r.rdb.Set(ctx, cacheKey, b, 30*time.Second).Err()
		}
	}

	return posts, total, nil
}

func (r *PostRepository) Create(ctx context.Context, uid uint, title, text string) (*models.Post, error) {
	slug, err := r.generateUniqueSlug(ctx, title)
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

func (r *PostRepository) CreateTx(ctx context.Context, tx *gorm.DB, uid uint, title, text string) (*models.Post, error) {
	slug, err := generateUniqueSlugWithDB(ctx, tx, title)
	if err != nil {
		return nil, err
	}

	post := &models.Post{
		Title:  title,
		Text:   text,
		Slug:   slug,
		UserID: uid,
	}

	if err := tx.WithContext(ctx).Create(post).Error; err != nil {
		return nil, err
	}

	// bumpListVersion: тут нюанс — он трогает Redis.
	// В проде bump делается тоже через outbox/событие, но сейчас оставим как есть или вынесем позже.
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
