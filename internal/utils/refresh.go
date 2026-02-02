package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"
)

func refreshTTL() time.Duration {
	if s := os.Getenv("JWT_REFRESH_TTL_H"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 720 * time.Hour
}

func NewRefreshToken() (plain, hashHex string, exp time.Time, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plain = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(plain))
	hashHex = hex.EncodeToString(h[:])
	exp = time.Now().Add(refreshTTL())
	return
}

func HashRefresh(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}
