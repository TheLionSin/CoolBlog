package services

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid refresh token")
	ErrToken              = errors.New("token error")
	ErrForbidden          = errors.New("access denied")
	ErrPostNotFound       = errors.New("post not found")
	ErrNoFieldsToUpdate   = errors.New("no fields to update")
)
