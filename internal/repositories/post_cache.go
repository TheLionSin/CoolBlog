package repositories

import (
	"go_blog/models"
	"time"

	"gorm.io/gorm"
)

type cachedPost struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	Slug      string    `json:"slug"`
	UserID    uint      `json:"user_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type cachedPostList struct {
	Total int64        `json:"total"`
	Posts []cachedPost `json:"posts"`
}

func toCachedPost(p models.Post) cachedPost {
	return cachedPost{
		ID:        p.ID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		Title:     p.Title,
		Text:      p.Text,
		Slug:      p.Slug,
		UserID:    p.UserID,
		IsActive:  p.IsActive,
	}
}

func (c cachedPost) toModel() *models.Post {
	return &models.Post{
		Model: gorm.Model{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		},
		Title:    c.Title,
		Text:     c.Text,
		Slug:     c.Slug,
		UserID:   c.UserID,
		IsActive: c.IsActive,
	}
}
