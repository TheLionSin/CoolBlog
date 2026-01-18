package models

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	PostID uint   `gorm:"index"`
	UserID uint   `gorm:"index"`
	Text   string `gorm:"type:text"`
}
