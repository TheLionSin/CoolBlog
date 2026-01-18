package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetPage(c *gin.Context) (page, limit int) {
	page = 1
	limit = 10
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	return
}

func Offset(page, limit int) int {
	return (page - 1) * limit
}
