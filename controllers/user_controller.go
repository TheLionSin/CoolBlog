package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go_blog/services"
	"go_blog/utils"
	"gorm.io/gorm"
	"net/http"
)

func GetCurrentUser(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := utils.MustUserID(c)
		if !ok {
			return
		}
		resp, err := userService.Me(c.Request.Context(), uid)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils.RespondError(c, http.StatusNotFound, "user not found")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "db error")
			return
		}

		utils.RespondOK(c, resp)
	}
}
