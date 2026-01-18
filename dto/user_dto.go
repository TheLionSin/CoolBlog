package dto

import "go_blog/models"

type UserResponse struct {
	ID       uint          `json:"id"`
	Nickname string        `json:"nickname"`
	Email    string        `json:"email"`
	Posts    []models.Post `json:"posts"`
}
