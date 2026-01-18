package models

import "gorm.io/gorm"

type Post struct {
	gorm.Model
	Title    string    `gorm:"size:150;not null"`
	Text     string    `gorm:"type:text"`
	Slug     string    `gorm:"size:200;uniqueIndex;not null"`
	UserID   uint      `gorm:"not null;index"`
	User     User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	IsActive bool      `gorm:"default:true"`
	Comments []Comment `gorm:"foreignKey:PostID"`
}
