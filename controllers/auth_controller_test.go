package controllers_test

import (
	"encoding/json"
	"go_blog/dto"
	"go_blog/testhelpers"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthController_Register_OK(t *testing.T) {
	app := testhelpers.SetupAuthTestApp(t)

	req := testhelpers.NewJSONRequest("POST", "/auth/register", dto.RegisterRequest{
		Nickname: "test",
		Email:    "reg@test.com",
		Password: "123456",
	})

	resp := testhelpers.DoRequest(app, req)

	require.Equal(t, http.StatusCreated, resp.Code)
}

func TestAuthController_Register_Duplicate(t *testing.T) {
	app := testhelpers.SetupAuthTestApp(t)

	req1 := testhelpers.NewJSONRequest("POST", "/auth/register", dto.RegisterRequest{
		Nickname: "test1",
		Email:    "dup@test.com",
		Password: "12345678",
	})
	_ = testhelpers.DoRequest(app, req1)

	req2 := testhelpers.NewJSONRequest("POST", "/auth/register", dto.RegisterRequest{
		Nickname: "test2",        // важно
		Email:    "dup@test.com", // тот же email
		Password: "12345678",
	})

	resp := testhelpers.DoRequest(app, req2)
	require.Equal(t, http.StatusConflict, resp.Code)
}

func TestAuthController_Login_OK(t *testing.T) {
	app := testhelpers.SetupAuthTestApp(t)

	_ = testhelpers.DoRequest(app,
		testhelpers.NewJSONRequest("POST", "/auth/register", dto.RegisterRequest{
			Nickname: "test",
			Email:    "login@test.com",
			Password: "123456",
		}),
	)

	resp := testhelpers.DoRequest(app,
		testhelpers.NewJSONRequest("POST", "/auth/login", dto.LoginRequest{
			Email:    "login@test.com",
			Password: "123456",
		}),
	)

	require.Equal(t, http.StatusOK, resp.Code)
}

func TestAuthController_Login_Invalid(t *testing.T) {
	app := testhelpers.SetupAuthTestApp(t)

	resp := testhelpers.DoRequest(app,
		testhelpers.NewJSONRequest("POST", "/auth/login", dto.LoginRequest{
			Email:    "no@test.com",
			Password: "21da21f",
		}),
	)

	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuthController_Refresh_OK(t *testing.T) {
	app := testhelpers.SetupAuthTestApp(t)

	// register
	testhelpers.DoRequest(app,
		testhelpers.NewJSONRequest("POST", "/auth/register", dto.RegisterRequest{
			Nickname: "test",
			Email:    "ref@test.com",
			Password: "123456",
		}),
	)

	// login
	loginResp := testhelpers.DoRequest(app,
		testhelpers.NewJSONRequest("POST", "/auth/login", dto.LoginRequest{
			Email:    "ref@test.com",
			Password: "123456",
		}),
	)
	require.Equal(t, http.StatusOK, loginResp.Code)

	type okResponse[T any] struct {
		Ok   bool `json:"ok"`
		Data T    `json:"data"`
	}

	var out okResponse[dto.TokenPairResponse]
	require.NoError(t, json.NewDecoder(loginResp.Body).Decode(&out))
	require.True(t, out.Ok)
	require.NotEmpty(t, out.Data.RefreshToken)

	// refresh
	refreshResp := testhelpers.DoRequest(app,
		testhelpers.NewJSONRequest("POST", "/auth/refresh", dto.RefreshTokenRequest{
			RefreshToken: out.Data.RefreshToken,
		}),
	)

	require.Equal(t, http.StatusOK, refreshResp.Code)
}

func TestAuthController_Logout_Idempotent(t *testing.T) {
	app := testhelpers.SetupAuthTestApp(t)

	resp := testhelpers.DoRequest(app,
		testhelpers.NewJSONRequest("POST", "/auth/logout", dto.RefreshTokenRequest{
			RefreshToken: "invalid",
		}),
	)

	require.Equal(t, http.StatusOK, resp.Code)
}
