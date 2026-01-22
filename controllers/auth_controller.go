package controllers

import (
	"go_blog/dto"
	"go_blog/services"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Register(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.RespondError(c, http.StatusBadRequest, "invalid json")
			return
		}

		if err := validators.Validate.Struct(req); err != nil {
			utils.RespondValidation(c, validationErrors(err))
			return
		}

		out, err := auth.Register(c.Request.Context(), req)
		if err != nil {
			if respondServiceError(c, err) {
				return
			}
			utils.RespondError(c, http.StatusInternalServerError, "register failed")
			return
		}

		utils.RespondCreated(c, out)
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
			utils.RespondValidation(c, validationErrors(err))
			return
		}

		out, err := auth.Login(c.Request.Context(), req)
		if err != nil {
			if respondServiceError(c, err) {
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
			utils.RespondValidation(c, validationErrors(err))
			return
		}

		out, err := auth.Refresh(c.Request.Context(), req.RefreshToken)
		if err != nil {
			if respondServiceError(c, err) {
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
