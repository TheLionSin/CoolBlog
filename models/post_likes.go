package models

import "gorm.io/gorm"

type PostLike struct {
	gorm.Model

	UserID uint `gorm:"not null;index;uniqueIndex:idx_user_post"`
	PostID uint `gorm:"not null;index;uniqueIndex:idx_user_post"`
}
