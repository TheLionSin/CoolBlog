package services

import (
	"context"
	"go_blog/dto"
	"go_blog/internal/repositories"
	"go_blog/models"
	"go_blog/stores"
	"go_blog/testhelpers"
	"go_blog/utils"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createUserViaService(t *testing.T, svc *AuthService, email, password string) uint {
	t.Helper()

	out, err := svc.Register(context.Background(), dto.RegisterRequest{
		Nickname: "test_" + email,
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	return out.ID
}

func getUser(t *testing.T, db *gorm.DB, id uint) *models.User {
	t.Helper()
	var u models.User
	require.NoError(t, db.First(&u, id).Error)
	return &u
}

func TestAuthService_Login_OK_And_Refresh_Works(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	rdb := testhelpers.SetupTestRedis(t)

	userRepo := repositories.NewUserRepository(db)
	tokenStore := stores.NewRefreshRedisStore(rdb)
	svc := NewAuthService(userRepo, tokenStore)

	email := "login_ok@test.com"
	pass := "12345s"

	_ = createUserViaService(t, svc, email, pass)

	tokens, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    email,
		Password: pass,
	})
	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)

	// refresh должен работать
	newTokens, err := svc.Refresh(context.Background(), tokens.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, newTokens.AccessToken)
	require.NotEmpty(t, newTokens.RefreshToken)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	rdb := testhelpers.SetupTestRedis(t)

	userRepo := repositories.NewUserRepository(db)
	tokenStore := stores.NewRefreshRedisStore(rdb)
	svc := NewAuthService(userRepo, tokenStore)

	email := "bad@test.com"
	pass := "12345s"
	_ = createUserViaService(t, svc, email, pass)

	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    email,
		Password: "wrong",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Refresh_Rotation_OldDies(t *testing.T) {

	db := testhelpers.SetupTestDB(t)
	rdb := testhelpers.SetupTestRedis(t)
	userRepo := repositories.NewUserRepository(db)
	tokenStore := stores.NewRefreshRedisStore(rdb)
	svc := NewAuthService(userRepo, tokenStore)

	email := "rot@test.com"
	pass := "123456"
	_ = createUserViaService(t, svc, email, pass)

	t1, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    email,
		Password: pass,
	})
	require.NoError(t, err)

	t2, err := svc.Refresh(context.Background(), t1.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, t1.RefreshToken, t2.RefreshToken)

	// старый refresh должен стать невалидным
	_, err = svc.Refresh(context.Background(), t1.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidRefresh)

}

func TestAuthService_MultiSession_LogoutOnlyOneSession(t *testing.T) {

	db := testhelpers.SetupTestDB(t)
	rdb := testhelpers.SetupTestRedis(t)

	userRepo := repositories.NewUserRepository(db)
	tokenStore := stores.NewRefreshRedisStore(rdb)
	svc := NewAuthService(userRepo, tokenStore)

	email := "ms@test.com"
	pass := "123456"
	_ = createUserViaService(t, svc, email, pass)

	t1, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    email,
		Password: pass,
	})
	require.NoError(t, err)

	t2, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    email,
		Password: pass,
	})
	require.NoError(t, err)
	require.NotEqual(t, t1.RefreshToken, t2.RefreshToken)

	// logout первой сессии
	require.NoError(t, svc.Logout(context.Background(), t1.RefreshToken))

	// t1 должен умереть
	_, err = svc.Refresh(context.Background(), t1.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidRefresh)

	// t2 должен жить
	_, err = svc.Refresh(context.Background(), t2.RefreshToken)
	require.NoError(t, err)

}

func TestAuthService_Logout_InvalidRefresh_IsOK(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	rdb := testhelpers.SetupTestRedis(t)

	userRepo := repositories.NewUserRepository(db)
	tokenStore := stores.NewRefreshRedisStore(rdb)
	svc := NewAuthService(userRepo, tokenStore)

	require.NoError(t, svc.Logout(context.Background(), "not-a-refresh-token"))
}

func TestAuthService_Refresh_InvalidRefresh(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	rdb := testhelpers.SetupTestRedis(t)

	userRepo := repositories.NewUserRepository(db)
	tokenStore := stores.NewRefreshRedisStore(rdb)
	svc := NewAuthService(userRepo, tokenStore)

	_, err := svc.Refresh(context.Background(), "not-a-refresh-token")
	require.ErrorIs(t, err, ErrInvalidRefresh)
}

func TestAuthService_Refresh_UsesCurrentUserRole(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	rdb := testhelpers.SetupTestRedis(t)

	userRepo := repositories.NewUserRepository(db)
	tokenStore := stores.NewRefreshRedisStore(rdb)
	svc := NewAuthService(userRepo, tokenStore)

	email := "role@test.com"
	pass := "12345s"
	uid := createUserViaService(t, svc, email, pass)

	// 1) логин → получаем refresh
	t1, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    email,
		Password: pass,
	})
	require.NoError(t, err)

	// 2) меняем роль в БД
	require.NoError(t,
		db.Model(&models.User{}).Where("id = ?", uid).Update("role", "admin").Error,
	)

	// 3) refresh → новый access должен содержать role=admin
	t2, err := svc.Refresh(context.Background(), t1.RefreshToken)
	require.NoError(t, err)

	token, claims, err := utils.ParseAccessJWT(t2.AccessToken)
	require.NoError(t, err)
	require.True(t, token.Valid)

	//role
	role, ok := claims["role"].(string)
	require.True(t, ok)
	require.Equal(t, "admin", role)

	// sub (uint приходит как float64)
	sub, ok := claims["sub"].(float64)
	require.True(t, ok)
	require.Equal(t, float64(uid), sub)
}
