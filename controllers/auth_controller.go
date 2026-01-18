package controllers

import (
	"errors"
	"fmt"
	"go_blog/config"
	"go_blog/dto"
	"go_blog/models"
	"go_blog/utils"
	"go_blog/validators"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid json")
		return
	}

	if err := validators.Validate.Struct(req); err != nil {
		errorsMap := make(map[string]string)
		for _, e := range err.(validator.ValidationErrors) {
			errorsMap[e.Field()] = fmt.Sprintf("не проходит '%s", e.Tag())
		}
		utils.RespondValidation(c, errorsMap)
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "hash error")
		return
	}

	user := models.User{
		Nickname: req.Nickname,
		Email:    req.Email,
		Password: hash,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "db error")
		return
	}

	resp := dto.RegisterResponse{
		ID:       user.ID,
		Nickname: user.Nickname,
		Email:    user.Email,
	}

	utils.RespondCreated(c, resp)
}

func Login(c *gin.Context) {
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

	var user models.User
	if err := config.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondError(c, http.StatusUnauthorized, "invalid credentials")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "db error")
		return
	}

	if !utils.CheckPasswordHash(user.Password, req.Password) {
		utils.RespondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	access, err := utils.GenerateAccessJWT(user.ID, user.Role)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "token error")
		return
	}

	plain, hash, exp, err := utils.NewRefreshToken()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "token error")
		return
	}

	ttl := time.Until(exp)
	ctx := c.Request.Context()

	tokenKey := utils.RefreshTokenKey(hash)
	userKey := utils.RefreshUserKey(user.ID)

	pipe := config.RDB.Pipeline()
	pipe.Set(ctx, tokenKey, user.ID, ttl)
	pipe.Set(ctx, userKey, hash, ttl)
	_, err = pipe.Exec(ctx)

	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "redis error")
		return
	}

	utils.RespondOK(c, dto.TokenPairResponse{
		AccessToken:  access,
		RefreshToken: plain,
	})
}

func Refresh(c *gin.Context) {
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

	hash := utils.HashRefresh(req.RefreshToken)
	ctx := c.Request.Context()

	tokenKey := utils.RefreshTokenKey(hash)

	userID, err := config.RDB.Get(ctx, tokenKey).Uint64()
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	var user models.User
	if err := config.DB.First(&user, uint(userID)).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "db error")
		return
	}

	access, err := utils.GenerateAccessJWT(user.ID, user.Role)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "token error")
		return
	}

	plain, newHash, exp, err := utils.NewRefreshToken()
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "token error")
		return
	}

	ttl := time.Until(exp)

	newTokenKey := utils.RefreshTokenKey(newHash)
	userKey := utils.RefreshUserKey(user.ID)

	pipe := config.RDB.Pipeline()
	pipe.Del(ctx, tokenKey, userKey)
	pipe.Set(ctx, newTokenKey, user.ID, ttl)
	pipe.Set(ctx, userKey, newHash, ttl)
	_, err = pipe.Exec(ctx)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "redis error")
		return
	}

	utils.RespondOK(c, dto.TokenPairResponse{
		AccessToken:  access,
		RefreshToken: plain,
	})
}

func Logout(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid json")
		return
	}

	ctx := c.Request.Context()
	hash := utils.HashRefresh(req.RefreshToken)

	tokenKey := utils.RefreshTokenKey(hash)

	userID, err := config.RDB.Get(ctx, tokenKey).Uint64()
	if err != nil {
		utils.RespondOK(c, gin.H{"message": "logged out"})
		return
	}

	userKey := utils.RefreshUserKey(uint(userID))

	_ = config.RDB.Del(ctx, tokenKey, userKey).Err()
	utils.RespondOK(c, gin.H{"message": "logged out"})
}

func GetCurrentUser(c *gin.Context) {
	uid, ok := c.Get("userID")
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var user models.User
	if err := config.DB.Select("id", "nickname", "email").Preload("Posts").First(&user, uid.(uint)).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, "user not found")
		return
	}

	utils.RespondOK(c, dto.UserResponse{
		ID:       user.ID,
		Nickname: user.Nickname,
		Email:    user.Email,
		Posts:    user.Posts,
	})
}
