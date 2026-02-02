package controllers

import (
	"errors"
	"go_blog/internal/services"
	utils2 "go_blog/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetCurrentUser(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := utils2.MustUserID(c)
		if !ok {
			return
		}
		resp, err := userService.Me(c.Request.Context(), uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils2.RespondError(c, http.StatusNotFound, "user not found")
				return
			}
			utils2.RespondError(c, http.StatusInternalServerError, "db error")
			return
		}

		utils2.RespondOK(c, resp)
	}
}
