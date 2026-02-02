package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func RefreshUserKey(userID uint) string {
	return fmt.Sprintf("refresh:user:%d", userID)
}

func RefreshTokenKey(hash string) string {
	return "refresh:token:" + hash
}

func PostsListVersionKey() string {
	return "posts:list:ver"
}

func PostsListsCacheKey(version int64, page, limit int, q string) string {
	q = strings.TrimSpace(strings.ToLower(q))
	sum := sha256.Sum256([]byte(q))
	qh := hex.EncodeToString(sum[:8])
	return fmt.Sprintf("posts:list:v%d:p%d:l%d:q%s", version, page, limit, qh)
}
