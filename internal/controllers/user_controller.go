package controllers

import (
	"errors"
	"fmt"
	"go_blog/internal/dto"
	"go_blog/internal/services"
	"go_blog/internal/utils"
	"go_blog/internal/validators"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type UserController struct {
	service *services.UserService
}

func NewUserController(service *services.UserService) *UserController {
	return &UserController{
		service: service,
	}
}

func (uc *UserController) GetMe(c *gin.Context) {
	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	resp, err := uc.service.GetMe(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.RespondError(c, http.StatusNotFound, "user not found")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "failed to get user profile")
		return
	}

	utils.RespondOK(c, resp)
}

func (uc *UserController) UpdateMe(c *gin.Context) {
	uid, ok := utils.MustUserID(c)
	if !ok {
		return
	}

	var req dto.UserUpdateReq
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

	resp, err := uc.service.UpdateProfile(c.Request.Context(), uid, req)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			utils.RespondError(c, http.StatusNotFound, "user not found")
		case errors.Is(err, services.ErrNoFieldsToUpdate):
			utils.RespondError(c, http.StatusBadRequest, "nothing to update")
		default:
			utils.RespondError(c, http.StatusInternalServerError, "failed to update profile")
		}
		return
	}

	utils.RespondOK(c, resp)
}
