package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Nickname string    `gorm:"size:30;not null;uniqueIndex:idx_users_nickname"`
	Email    string    `gorm:"size:255;not null;uniqueIndex:idx_users_email"`
	Password string    `gorm:"not null"`
	Posts    []Post    `gorm:"foreignKey:UserID"`
	Comments []Comment `gorm:"foreignKey:UserID"`
	IsActive bool      `gorm:"default:true"`
	Role     string    `gorm:"size:20;default:'user'"`
}
