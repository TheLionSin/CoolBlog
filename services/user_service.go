package services

import (
	"context"
	"go_blog/dto"
	"go_blog/internal/repositories"
)

type UserService struct {
	users *repositories.UserRepository
}

func NewUserService(users *repositories.UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) Me(ctx context.Context, userID uint) (dto.UserMeResponse, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return dto.UserMeResponse{}, err
	}

	return dto.UserMeResponse{
		ID:       u.ID,
		Nickname: u.Nickname,
		Email:    u.Email,
	}, nil
}
