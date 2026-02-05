package services

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid refresh token")
	ErrToken              = errors.New("token error")
	ErrPostNotFound       = errors.New("post not found")
	ErrNoFieldsToUpdate   = errors.New("no fields to update")
)
