package services

import (
	"context"
	"go_blog/dto"
	"go_blog/models"
	"go_blog/stores"
	"go_blog/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeUserRepo struct {
	users  map[uint]*models.User
	byMail map[string]*models.User
	nextID uint
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:  make(map[uint]*models.User),
		byMail: make(map[string]*models.User),
		nextID: 1,
	}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *models.User) error {
	u.ID = f.nextID
	f.nextID++
	f.users[u.ID] = u
	f.byMail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	u, ok := f.byMail[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id uint) (*models.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

type fakeRefreshStore struct {
	hashToUser map[string]uint
}

func newFakeRefreshStore() *fakeRefreshStore {
	return &fakeRefreshStore{
		hashToUser: make(map[string]uint),
	}
}

func (f *fakeRefreshStore) Save(ctx context.Context, uid uint, hash string, _ time.Duration) error {
	f.hashToUser[hash] = uid
	return nil
}

func (f *fakeRefreshStore) GetUserIDByHash(ctx context.Context, hash string) (uint, error) {
	uid, ok := f.hashToUser[hash]
	if !ok {
		return 0, stores.ErrInvalidRefresh
	}
	return uid, nil
}

func (f *fakeRefreshStore) Rotate(ctx context.Context, oldHash string, uid uint, newHash string, _ time.Duration) error {
	delete(f.hashToUser, oldHash)
	f.hashToUser[newHash] = uid
	return nil
}

func (f *fakeRefreshStore) Delete(ctx context.Context, hash string, uid uint) error {
	delete(f.hashToUser, hash)
	return nil
}

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

func TestAuthService_Login_OK_And_Refresh_Works(t *testing.T) {
	users := newFakeUserRepo()
	tokens := newFakeRefreshStore()
	svc := NewAuthService(users, tokens)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Nickname: "test",
		Email:    "a@test.com",
		Password: "123456",
	})
	require.NoError(t, err)

	t1, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    "a@test.com",
		Password: "123456",
	})
	require.NoError(t, err)

	t2, err := svc.Refresh(context.Background(), t1.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, t1.RefreshToken, t2.RefreshToken)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	users := newFakeUserRepo()
	tokens := newFakeRefreshStore()
	svc := NewAuthService(users, tokens)

	_, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    "no@test.com",
		Password: "wrong",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Refresh_Rotation_OldDies(t *testing.T) {
	users := newFakeUserRepo()
	tokens := newFakeRefreshStore()
	svc := NewAuthService(users, tokens)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Nickname: "test",
		Email:    "rot@test.com",
		Password: "123456",
	})
	require.NoError(t, err)

	t1, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    "rot@test.com",
		Password: "123456",
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
	users := newFakeUserRepo()
	tokens := newFakeRefreshStore()
	svc := NewAuthService(users, tokens)

	_, _ = svc.Register(context.Background(), dto.RegisterRequest{
		Nickname: "u",
		Email:    "ms@test.com",
		Password: "123456",
	})

	t1, _ := svc.Login(context.Background(), dto.LoginRequest{
		Email: "ms@test.com", Password: "123456",
	})
	t2, _ := svc.Login(context.Background(), dto.LoginRequest{
		Email: "ms@test.com", Password: "123456",
	})

	require.NoError(t, svc.Logout(context.Background(), t1.RefreshToken))

	_, err := svc.Refresh(context.Background(), t1.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidRefresh)

	_, err = svc.Refresh(context.Background(), t2.RefreshToken)
	require.NoError(t, err)
}

func TestAuthService_Logout_InvalidRefresh_IsOK(t *testing.T) {
	users := newFakeUserRepo()
	tokens := newFakeRefreshStore()
	svc := NewAuthService(users, tokens)

	// logout должен быть идемпотентным
	require.NoError(t, svc.Logout(context.Background(), "not-a-refresh-token"))
}

func TestAuthService_Refresh_InvalidRefresh(t *testing.T) {
	users := newFakeUserRepo()
	tokens := newFakeRefreshStore()
	svc := NewAuthService(users, tokens)

	_, err := svc.Refresh(context.Background(), "not-a-refresh-token")
	require.ErrorIs(t, err, ErrInvalidRefresh)
}

func TestAuthService_Refresh_UsesCurrentUserRole(t *testing.T) {
	users := newFakeUserRepo()
	tokens := newFakeRefreshStore()
	svc := NewAuthService(users, tokens)

	out, _ := svc.Register(context.Background(), dto.RegisterRequest{
		Nickname: "u",
		Email:    "role@test.com",
		Password: "123456",
	})

	t1, _ := svc.Login(context.Background(), dto.LoginRequest{
		Email: "role@test.com", Password: "123456",
	})

	users.users[out.ID].Role = "admin"

	t2, _ := svc.Refresh(context.Background(), t1.RefreshToken)

	_, claims, _ := utils.ParseAccessJWT(t2.AccessToken)
	require.Equal(t, "admin", claims["role"])
}
