package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go_blog/internal/dto"
	"go_blog/internal/services"
	"go_blog/internal/utils"
	"go_blog/internal/validators"
	"net/http"
)

// 1. Объявляем интерфейс (Чего мы хотим от сервиса)
// Это контракт. Контроллеру плевать, кто его исполняет.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest, userAgent, ip string) (*dto.TokenPairResponse, error)
	Login(ctx context.Context, req dto.LoginRequest, userAgent, ip string) (*dto.TokenPairResponse, error)
	Refresh(ctx context.Context, refreshToken string, userAgent, ip string) (*dto.TokenPairResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}
type AuthController struct {
	service AuthService
}

func NewAuthController(service AuthService) *AuthController {
	return &AuthController{service: service}
}

func (ac *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid json")
		return
	}

	if err := validators.Validate.Struct(req); err != nil {
		errorsMap := make(map[string]string)
		for _, e := range err.(validator.ValidationErrors) {
			errorsMap[e.Field()] = fmt.Sprintf("failed validation: %s", e.Tag())
		}
		utils.RespondValidation(c, errorsMap)
		return
	}

	//Get metadata
	userAgent := c.Request.UserAgent()
	ip := c.ClientIP()

	tokens, err := ac.service.Register(c.Request.Context(), req, userAgent, ip)
	if err != nil {
		if errors.Is(err, services.ErrUserExists) {
			utils.RespondError(c, http.StatusBadRequest, "user already exists")
		} else {
			utils.RespondError(c, http.StatusInternalServerError, "registration failed")
		}
		return
	}

	utils.RespondCreated(c, tokens)
}

func (ac *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid json")
		return
	}

	if err := validators.Validate.Struct(req); err != nil {
		errorsMap := make(map[string]string)
		for _, e := range err.(validator.ValidationErrors) {
			errorsMap[e.Field()] = fmt.Sprintf("failed validation: %s", e.Tag())
		}
		utils.RespondValidation(c, errorsMap)
		return
	}

	tokens, err := ac.service.Login(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.RespondError(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "login failed")
		return
	}

	utils.RespondOK(c, tokens)
}

func (ac *AuthController) Refresh(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid json")
		return
	}

	tokens, err := ac.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	utils.RespondOK(c, tokens)
}

func (ac *AuthController) Logout(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid json")
		return
	}

	_ = ac.service.Logout(c.Request.Context(), req.RefreshToken)
	utils.RespondOK(c, gin.H{"message": "logged out"})
}
