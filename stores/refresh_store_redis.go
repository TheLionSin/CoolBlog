package stores

import (
	"context"
	"errors"
	"go_blog/utils"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrInvalidRefresh = errors.New("invalid refresh token")

type RefreshStore interface {
	Save(ctx context.Context, userID uint, hash string, ttl time.Duration) error
	GetUserIDByHash(ctx context.Context, hash string) (uint, error)
	Rotate(ctx context.Context, oldHash string, userID uint, newHash string, ttl time.Duration) error
	Delete(ctx context.Context, hash string, userID uint) error
}

type RefreshRedisStore struct {
	rdb *redis.Client
}

func NewRefreshRedisStore(rdb *redis.Client) *RefreshRedisStore {
	return &RefreshRedisStore{rdb: rdb}
}

// refresh:token:<hash> -> userID
// refresh:user:<id>    -> hash

func (s *RefreshRedisStore) Save(ctx context.Context, userID uint, hash string, ttl time.Duration) error {
	uk := utils.RefreshUserKey(userID)

	oldHash, err := s.rdb.Get(ctx, uk).Result()
	if err == nil && oldHash != "" {
		_ = s.rdb.Del(ctx)
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, utils.RefreshTokenKey(hash), userID, ttl)
	pipe.Set(ctx, uk, hash, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RefreshRedisStore) GetUserIDByHash(ctx context.Context, hash string) (uint, error) {
	u64, err := s.rdb.Get(ctx, utils.RefreshTokenKey(hash)).Uint64()
	if err != nil {
		return 0, ErrInvalidRefresh
	}
	return uint(u64), nil
}

func (s *RefreshRedisStore) Rotate(ctx context.Context, oldHash string, userID uint, newHash string, ttl time.Duration) error {
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, utils.RefreshTokenKey(oldHash))              // убрать старый tokenKey
	pipe.Set(ctx, utils.RefreshTokenKey(newHash), userID, ttl) // новый tokenKey
	pipe.Set(ctx, utils.RefreshUserKey(userID), newHash, ttl)  // обновить userKey
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RefreshRedisStore) Delete(ctx context.Context, hash string, userID uint) error {
	tk := utils.RefreshTokenKey(hash)
	uk := utils.RefreshUserKey(userID)

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, tk)

	cur, err := s.rdb.Get(ctx, uk).Result()
	if err == nil && cur == hash {
		pipe.Del(ctx, uk)
	}

	_, err = pipe.Exec(ctx)
	return err
}
