package utils

import (
	"fmt"
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
