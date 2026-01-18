package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func MustUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("userID")
	if !ok {
		RespondError(c, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	uid, ok := v.(uint)
	if !ok || uid == 0 {
		RespondError(c, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	return uid, true
}
