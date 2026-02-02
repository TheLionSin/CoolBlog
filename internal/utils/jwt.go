package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func jwtSecret() []byte {
	sec := os.Getenv("JWT_SECRET")
	return []byte(sec)
}

func accessTTL() time.Duration {
	if s := os.Getenv("JWT_ACCESS_TTL_MIN"); s != "" {
		if d, err := time.ParseDuration(s + "m"); err != nil {
			return d
		}
	}
	return 60 * time.Minute
}

func GenerateAccessJWT(userID uint, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(accessTTL()).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecret())
}

func ParseAccessJWT(tokenStr string) (*jwt.Token, jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	return token, claims, err
}
