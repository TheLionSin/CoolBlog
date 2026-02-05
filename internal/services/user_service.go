package services

import (
	"context"
	"go_blog/internal/dto"
	"go_blog/internal/events"
	"go_blog/internal/repositories"

	"gorm.io/gorm"
)

type UserService struct {
	db         *gorm.DB
	userRepo   *repositories.UserRepository
	outboxRepo *repositories.OutboxRepository
}

func NewUserService(db *gorm.DB, users *repositories.UserRepository, outboxRepo *repositories.OutboxRepository) *UserService {
	return &UserService{
		db:         db,
		userRepo:   users,
		outboxRepo: outboxRepo,
	}
}

// GetMe - просто получение профиля
func (s *UserService) GetMe(ctx context.Context, userID uint) (*dto.UserMeResponse, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.UserMeResponse{
		ID:       u.ID,
		Nickname: u.Nickname,
		Email:    u.Email,
		Role:     u.Role,
	}, nil
}

// UpdateProfile - транзакционное обновление
func (s *UserService) UpdateProfile(ctx context.Context, userID uint, req dto.UserUpdateReq) (*dto.UserMeResponse, error) {
	var response *dto.UserMeResponse

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Ищем юзера (чтобы убедиться, что он есть и получить текущие данные)
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil {
			return err
		}

		// 2. Готовим обновления
		updates := make(map[string]any)
		if req.Nickname != nil {
			updates["nickname"] = *req.Nickname
		}
		// Email менять сложнее (надо подтверждать), пока опустим, не забыть!
		if len(updates) == 0 {
			// Если обновлять нечего, просто возвращаем текущего
			response = &dto.UserMeResponse{
				ID:       user.ID,
				Nickname: user.Nickname,
				Email:    user.Email,
				Role:     user.Role,
			}
			return nil
		}

		//Обновляем в БД
		if err := s.userRepo.UpdateTx(ctx, tx, user, updates); err != nil {
			return err
		}

		if req.Nickname != nil {
			user.Nickname = *req.Nickname
		}

		//Событие
		evt, err := events.NewUserUpdatedEvent(user)
		if err != nil {
			return err
		}

		//Outbox
		if err := s.outboxRepo.CreateTx(ctx, tx, evt); err != nil {
			return err
		}

		response = &dto.UserMeResponse{
			ID:       user.ID,
			Nickname: user.Nickname,
			Email:    user.Email,
			Role:     user.Role,
		}
		return nil
	})

	return response, err
}
