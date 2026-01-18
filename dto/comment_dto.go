package dto

import "time"

type CommentCreateRequest struct {
	Text string `json:"text" validate:"required"`
}

type CommentResponse struct {
	ID        uint      `json:"id"`
	Text      string    `json:"text"`
	PostID    uint      `json:"post_id"`
	UserID    uint      `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
