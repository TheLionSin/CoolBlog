package middleware

import (
	"go_blog/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "missing bearer token"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		token, claims, err := utils.ParseAccessJWT(tokenStr)
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid token"})
			c.Abort()
			return
		}
		uid, ok := claims["sub"].(float64)
		if !ok || uid <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid subject"})
			c.Abort()
			return
		}

		role, _ := claims["role"].(string)

		c.Set("userID", uint(uid))
		c.Set("role", role)
		c.Next()
	}
}
