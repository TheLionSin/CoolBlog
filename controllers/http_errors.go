package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go_blog/internal/repositories"
	"go_blog/services"
	"go_blog/utils"
	"gorm.io/gorm"
	"net/http"
)

func respondServiceError(c *gin.Context, err error, fallbackMsg string) bool {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		utils.RespondError(c, http.StatusUnauthorized, "invalid credentials")
		return true

	case errors.Is(err, services.ErrInvalidRefresh):
		utils.RespondError(c, http.StatusUnauthorized, "invalid refresh token")
		return true

	case errors.Is(err, repositories.ErrUserExists):
		utils.RespondError(c, http.StatusConflict, "user already exists")
		return true

	case errors.Is(err, gorm.ErrRecordNotFound):
		utils.RespondError(c, http.StatusNotFound, "not found")
		return true
	}

	return false

}
