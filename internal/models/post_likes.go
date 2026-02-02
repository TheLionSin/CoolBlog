package models

import "time"

type PostLike struct {
	ID        uint `gorm:"primary_key"`
	UserID    uint `gorm:"not null;index;uniqueIndex:idx_user_post_unique"`
	PostID    uint `gorm:"not null;index;uniqueIndex:idx_user_post_unique"`
	CreatedAt time.Time
}
