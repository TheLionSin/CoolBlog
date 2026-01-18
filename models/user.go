package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Nickname string    `gorm:"size:30;not null;uniqueIndex"`
	Email    string    `gorm:"uniqueIndex;size:255;not null"`
	Password string    `gorm:"not null"`
	Posts    []Post    `gorm:"foreignKey:UserID"`
	Comments []Comment `gorm:"foreignKey:UserID"`
	IsActive bool      `gorm:"default:true"`
	Role     string    `gorm:"size:20;default:'user'"`
}
