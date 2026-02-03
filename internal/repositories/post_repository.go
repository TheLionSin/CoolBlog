package repositories

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go_blog/internal/models"
	utils2 "go_blog/internal/utils"
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

	base := utils2.Slugify(title)
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

	key := utils2.PostsListVersionKey()

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
	_ = r.rdb.Incr(ctx, utils2.PostsListVersionKey()).Err()
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
		Select("*").
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
	offset := utils2.Offset(page, limit)
	if err := db.
		Select("*").
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
	baseSlug := utils2.Slugify(title)
	slug := baseSlug

	// Пытаемся вставить в цикле
	for i := 0; i < 10; i++ { // 10 попыток максимум, чтобы не уйти в вечный цикл
		post := &models.Post{
			Title:  title,
			Text:   text,
			Slug:   slug,
			UserID: uid,
		}

		// 1. Делаем точку сохранения перед опасной операцией
		tx.SavePoint("sp_slug_check")

		// 2. Пробуем создать
		err := tx.WithContext(ctx).Create(post).Error
		if err == nil {
			r.bumpListVersion(ctx)
			return post, nil
		}

		// 3. ЕСЛИ ОШИБКА: Обязательно откатываемся к точке сохранения!
		// Это "оживляет" транзакцию в Postgres, отменяя последнюю неудачную команду.
		tx.RollbackTo("sp_slug_check")

		// 4. Анализируем ошибку
		// Приводим к нижнему регистру для надежности
		errStr := strings.ToLower(err.Error())

		// Ищем "unique" (для SQLite/других баз), "duplicate" (Postgres) или код "23505"
		if strings.Contains(errStr, "unique") ||
			strings.Contains(errStr, "duplicate") ||
			strings.Contains(errStr, "23505") {

			// Это ошибка уникальности. Генерируем новый слаг и пробуем снова.
			// В логах ты все равно будет красная ошибка
			// это GORM ругается до того, как мы обработали ошибку.
			slug = fmt.Sprintf("%s-%d", baseSlug, i+1)
			continue
		}

		// Если ошибка другая (например, база упала) - возвращаем её
		return nil, err
	}

	return nil, fmt.Errorf("failed to generate unique slug after retries")
}

func (r *PostRepository) UpdateTx(ctx context.Context, tx *gorm.DB, slug string, uid uint, updates map[string]any) (*models.Post, error) {
	var post models.Post
	// 1. Ищем пост (блокируем строку FOR UPDATE, чтобы избежать гонок при редактировании)
	// Clause(clause.Locking{Strength: "UPDATE"}) - это уровень Senior, пока можно без него, но полезно знать
	if err := tx.WithContext(ctx).Where("slug = ? AND user_id = ? AND is_active = ?", slug, uid, true).First(&post).Error; err != nil {
		return nil, err
	}

	//2. Обновляем
	if err := tx.WithContext(ctx).Model(&post).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 3. Возвращаем обновленный объект (нужен для события)
	if err := tx.WithContext(ctx).First(&post, post.ID).Error; err != nil {
		return nil, err
	}

	// 4. Сбрасываем кэши (Redis чистим тут же, так как это не отменить)
	// (По-хорошему кэш надо чистить ПОСЛЕ коммита транзакции, но пока допустимо тут)
	if r.rdb != nil {
		_ = r.rdb.Del(ctx, postBySlugKey(post.Slug)).Err()
	}
	// r.bumpListVersion(ctx) - это лучше вызывать в сервисе или оставить тут, но помнить про Redis

	return &post, nil
}

func (r *PostRepository) DeleteTx(ctx context.Context, tx *gorm.DB, slug string, uid uint) (*models.Post, error) {
	var post models.Post
	if err := tx.WithContext(ctx).Where("slug = ? AND user_id = ? AND is_active = ?", slug, uid, true).First(&post).Error; err != nil {
		return nil, err
	}

	if err := tx.WithContext(ctx).Delete(&post).Error; err != nil {
		return nil, err
	}
	if r.rdb != nil {
		_ = r.rdb.Del(ctx, postBySlugKey(slug)).Err()
	}

	return &post, nil
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
