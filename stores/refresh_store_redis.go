package stores

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"go_blog/utils"
	"time"
)

var ErrInvalidRefresh = errors.New("invalid refresh token")

type RefreshRedisStore struct {
	rdb *redis.Client
}

func NewRefreshRedisStore(rdb *redis.Client) *RefreshRedisStore {
	return &RefreshRedisStore{rdb: rdb}
}

// refresh:token:<hash> -> userID
// refresh:user:<id>    -> hash

func (s *RefreshRedisStore) Save(ctx context.Context, userID uint, hash string, ttl time.Duration) error {
	tokenKey := utils.RefreshTokenKey(hash)
	userKey := utils.RefreshUserKey(userID)

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, tokenKey, userID, ttl)
	pipe.Set(ctx, userKey, hash, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RefreshRedisStore) GetUserIDByHash(ctx context.Context, hash string) (uint, error) {
	tokenKey := utils.RefreshTokenKey(hash)

	uid, err := s.rdb.Get(ctx, tokenKey).Uint64()
	if err != nil {
		return 0, ErrInvalidRefresh
	}
	return uint(uid), nil
}

func (s *RefreshRedisStore) Rotate(ctx context.Context, oldHash string, userID uint, newHash string, ttl time.Duration) error {
	oldTokenKey := utils.RefreshTokenKey(oldHash)
	userKey := utils.RefreshUserKey(userID)
	newTokenKey := utils.RefreshTokenKey(newHash)

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, oldTokenKey)
	pipe.Set(ctx, newTokenKey, userID, ttl)
	pipe.Set(ctx, userKey, newHash, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RefreshRedisStore) RevokeByHash(ctx context.Context, hash string) error {
	tokenKey := utils.RefreshTokenKey(hash)

	uid, err := s.rdb.Get(ctx, tokenKey).Uint64()
	if err != nil {
		return nil
	}
	userKey := utils.RefreshUserKey(uint(uid))

	return s.rdb.Del(ctx, userKey).Err()
}
