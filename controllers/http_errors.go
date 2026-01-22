package controllers

import (
	"errors"
	"go_blog/internal/repositories"
	"go_blog/services"
	"go_blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func respondServiceError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		utils.RespondError(c, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, services.ErrInvalidRefresh):
		utils.RespondError(c, http.StatusUnauthorized, "invalid refresh token")
	case errors.Is(err, repositories.ErrUserExists):
		utils.RespondError(c, http.StatusConflict, "user already exists")
	case errors.Is(err, gorm.ErrRecordNotFound):
		utils.RespondError(c, http.StatusNotFound, "not found")
	default:
		return false
	}
	return true
}
