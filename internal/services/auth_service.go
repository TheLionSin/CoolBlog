package services

import (
	"context"
	"errors"
	"go_blog/internal/dto"
	"go_blog/internal/events"
	"go_blog/internal/models"
	"go_blog/internal/repositories"
	"go_blog/internal/utils"
	"time"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateTx(ctx context.Context, tx *gorm.DB, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	// FindByID может пригодиться позже
}

type TokenRepository interface {
	CreateTx(ctx context.Context, tx *gorm.DB, token *models.RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*models.RefreshToken, error)
	Delete(ctx context.Context, id uint) error
}

type OutboxRepository interface {
	CreateTx(ctx context.Context, tx *gorm.DB, evt *models.OutboxEvent) error
}

type AuthService struct {
	db         *gorm.DB
	userRepo   UserRepository
	tokenRepo  TokenRepository
	outboxRepo OutboxRepository
}

func NewAuthService(db *gorm.DB,
	userRepo UserRepository,
	tokenRepo TokenRepository,
	outboxRepo OutboxRepository,
) *AuthService {
	return &AuthService{
		db:         db,
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		outboxRepo: outboxRepo,
	}
}

// Register - Атомарная регистрация + Вход + Событие
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest, userAgent, ip string) (*dto.TokenPairResponse, error) {
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Nickname: req.Nickname,
		Email:    req.Email,
		Password: hash,
		Role:     "user",
		IsActive: true,
	}

	var tokens *dto.TokenPairResponse

	// START TRANSACTION
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//1. Create user
		if err := s.userRepo.CreateTx(ctx, tx, user); err != nil {
			if errors.Is(err, repositories.ErrUserExists) {
				//ПРОВЕРИТЬ
				return services.ErrUserExists
			}
			return err
		}

		//2. Generate tokens(auto-login)
		accessToken, err := utils.GenerateAccessJWT(user.ID, user.Role)
		if err != nil {
			return err
		}

		refreshPlain, refreshHash, exp, err := utils.NewRefreshToken()
		if err != nil {
			return err
		}

		//3. Save RefreshToken in DB(with user)
		tokenModel := &models.RefreshToken{
			UserID:    user.ID,
			TokenHash: refreshHash,
			ExpiresAt: exp,
			UserAgent: userAgent,
			IP:        ip,
		}
		if err := s.tokenRepo.CreateTx(ctx, tx, tokenModel); err != nil {
			return err
		}

		//4. Event "UserRegistered" (для email рассылки)
		evt, err := events.NewUserRegisteredEvent(user)
		if err != nil {
			return err
		}
		if err := s.outboxRepo.CreateTx(ctx, tx, evt); err != nil {
			return err
		}

		tokens = &dto.TokenPairResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshPlain, // Отдаем юзеру чистый токен (в базе только хэш)
		}
		return nil
	})

	return tokens, err

}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, userAgent, ip string) (*dto.TokenPairResponse, error) {

	//1. FindUser
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	//2. Check password
	if !utils.CheckPasswordHash(user.Password, req.Password) {
		return nil, ErrInvalidCredentials
	}

	//3. Generate tokens
	accessToken, err := utils.GenerateAccessJWT(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	refreshPlain, refreshHash, exp, err := utils.NewRefreshToken()
	if err != nil {
		return nil, err
	}

	// 4. Сохраняем сессию (Транзакция тут не обязательна, но желательна, если будем писать AuditLog входа)
	tokenModel := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: exp,
		UserAgent: userAgent,
		IP:        ip,
	}

	// Используем db.Transaction для консистентности (вдруг захотим добавить AuditLog "UserLoggedIn")
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.tokenRepo.CreateTx(ctx, tx, tokenModel)
	})
	if err != nil {
		return nil, err
	}

	return &dto.TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshPlain,
	}, nil

}

// Refresh - Обновление токенов (Rotate)
func (s *AuthService) Refresh(ctx context.Context, refreshPlain string, userAgent, ip string) (*dto.TokenPairResponse, error) {
	oldHash := utils.HashRefresh(refreshPlain)

	//1. Find token in db
	oldToken, err := s.tokenRepo.GetByHash(ctx, oldHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	//2. Check validate
	if oldToken.ExpiresAt.Before(time.Now()) {
		_ = s.tokenRepo.Delete(ctx, oldToken.ID) //Чистим протухший
		return nil, ErrInvalidToken
	}

	//3. Generate new
	user := oldToken.User
	accessToken, err := utils.GenerateAccessJWT(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	newPlain, newHash, exp, err := utils.NewRefreshToken()
	if err != nil {
		return nil, err
	}

	// 4. Ротация (Удаляем старый, создаем новый)
	// Важно делать в транзакции!

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//Delete old
		if err := tx.Delete(&models.RefreshToken{}, oldToken.ID).Error; err != nil {
			return err
		}

		//Create new
		newToken := &models.RefreshToken{
			UserID:    user.ID,
			TokenHash: newHash,
			ExpiresAt: exp,
			UserAgent: userAgent,
			IP:        ip,
		}
		return s.tokenRepo.CreateTx(ctx, tx, newToken)
	})

	if err != nil {
		return nil, err
	}

	return &dto.TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: newPlain,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshPlain string) error {
	hash := utils.HashRefresh(refreshPlain)
	token, err := s.tokenRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil // Уже удален или не найден, для логаута это ОК
	}

	return s.tokenRepo.Delete(ctx, token.ID)
}
