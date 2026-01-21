package services

import (
	"context"
	"errors"
	"go_blog/dto"
	"go_blog/internal/repositories"
	"go_blog/models"
	"go_blog/stores"
	"go_blog/utils"
	"gorm.io/gorm"
	"time"
)

type AuthService struct {
	users  *repositories.UserRepository
	tokens *stores.RefreshRedisStore
}

func NewAuthService(users *repositories.UserRepository, tokens *stores.RefreshRedisStore) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (dto.RegisterResponse, error) {
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return dto.RegisterResponse{}, err
	}

	user := models.User{
		Nickname: req.Nickname,
		Email:    req.Email,
		Password: hash,
		IsActive: true,
		Role:     "user"}

	if err := s.users.Create(ctx, &user); err != nil {
		return dto.RegisterResponse{}, err
	}

	return dto.RegisterResponse{ID: user.ID, Nickname: user.Nickname, Email: user.Email}, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (dto.TokenPairResponse, error) {
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.TokenPairResponse{}, ErrInvalidCredentials
		}
		return dto.TokenPairResponse{}, err
	}

	if !utils.CheckPasswordHash(user.Password, req.Password) {
		return dto.TokenPairResponse{}, ErrInvalidCredentials
	}

	access, err := utils.GenerateAccessJWT(user.ID, user.Role)
	if err != nil {
		return dto.TokenPairResponse{}, ErrToken
	}

	plain, hash, exp, err := utils.NewRefreshToken()
	if err != nil {
		return dto.TokenPairResponse{}, ErrToken
	}

	ttl := time.Until(exp)
	if err := s.tokens.Save(ctx, user.ID, hash, ttl); err != nil {
		return dto.TokenPairResponse{}, err
	}

	return dto.TokenPairResponse{
		AccessToken:  access,
		RefreshToken: plain,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshPlain string) (dto.TokenPairResponse, error) {
	oldHash := utils.HashRefresh(refreshPlain)

	uid, err := s.tokens.GetUserIDByHash(ctx, oldHash)
	if err != nil {
		return dto.TokenPairResponse{}, ErrInvalidRefresh
	}

	user, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return dto.TokenPairResponse{}, err
	}

	access, err := utils.GenerateAccessJWT(user.ID, user.Role)
	if err != nil {
		return dto.TokenPairResponse{}, ErrToken
	}

	plain, newHash, exp, err := utils.NewRefreshToken()
	if err != nil {
		return dto.TokenPairResponse{}, ErrToken
	}

	ttl := time.Until(exp)

	if err := s.tokens.Rotate(ctx, oldHash, user.ID, newHash, ttl); err != nil {
		return dto.TokenPairResponse{}, err
	}

	return dto.TokenPairResponse{
		AccessToken:  access,
		RefreshToken: plain,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshPlain string) error {
	hash := utils.HashRefresh(refreshPlain)
	return s.tokens.RevokeByHash(ctx, hash)
}
