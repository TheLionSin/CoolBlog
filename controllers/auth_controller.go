package controllers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go_blog/dto"
	"go_blog/services"
	"go_blog/stores"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"
)

func Register(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "invalid json")
			return
		}

		if err := validators.Validate.Struct(req); err != nil {
			errorsMap := make(map[string]string)
			for _, e := range err.(validator.ValidationErrors) {
				errorsMap[e.Field()] = fmt.Sprintf("не проходит '%s'", e.Tag())
			}
			utils.RespondValidation(c, errorsMap)
			return
		}
		resp, err := auth.Register(c.Request.Context(), req)
		if err != nil {
			if respondServiceError(c, err, "register failed") {
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "db error")
			return
		}

		utils.RespondCreated(c, resp)
	}
}

func Login(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadGateway, "invalid json")
			return
		}

		if err := validators.Validate.Struct(req); err != nil {
			errorsMap := make(map[string]string)
			for _, e := range err.(validator.ValidationErrors) {
				errorsMap[e.Field()] = fmt.Sprintf("не проходит '%s'", e.Tag())
			}
			utils.RespondValidation(c, errorsMap)
			return
		}

		out, err := auth.Login(c.Request.Context(), req)
		if err != nil {
			if respondServiceError(c, err, "login failed") {
				return
			}

			utils.RespondError(c, http.StatusInternalServerError, "login failed")
			return
		}

		utils.RespondOK(c, out)

	}
}

func Refresh(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {

		var req dto.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "invalid json")
			return
		}

		if err := validators.Validate.Struct(req); err != nil {
			errorsMap := map[string]string{}
			for _, e := range err.(validator.ValidationErrors) {
				errorsMap[e.Field()] = fmt.Sprintf("не проходит '%s'", e.Tag())
			}
			utils.RespondValidation(c, errorsMap)
			return
		}

		out, err := auth.Refresh(c.Request.Context(), req.RefreshToken)
		if err != nil {
			if errors.Is(err, stores.ErrInvalidRefresh) {
				utils.RespondError(c, http.StatusUnauthorized, "invalid refresh token")
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "refresh failed")
			return
		}

		utils.RespondOK(c, out)
	}
}

func Logout(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "invalid json")
			return
		}

		_ = auth.Logout(c.Request.Context(), req.RefreshToken)
		utils.RespondOK(c, gin.H{"message": "logged out"})
	}
}
