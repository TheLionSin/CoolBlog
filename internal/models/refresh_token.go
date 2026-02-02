package models

import (
	"time"
)

type RefreshToken struct {
	ID        uint      `gorm:"primary_key"`
	UserID    uint      `gorm:"index"`
	TokenHash string    `gorm:"uniqueIndex;size:64;not null"`
	CreatedAt time.Time `gorm:"index"`
	ExpiresAt time.Time `gorm:"index"`
	UserAgent string    `gorm:"size:255"`
	IP        string    `gorm:"size:45"`
}
