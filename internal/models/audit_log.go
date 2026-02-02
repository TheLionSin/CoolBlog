package models

import "time"

type AuditLog struct {
	ID            uint      `gorm:"primaryKey"`
	EventID       string    `gorm:"size:36;uniqueIndex;not null"`
	EventType     string    `gorm:"size:50;not null"`
	AggregateType string    `gorm:"size:50;not null"`
	AggregateID   string    `gorm:"size:50;not null"`
	ActorUserID   string    `gorm:"size:50"`
	Payload       string    `gorm:"type:jsonb;not null"`
	OccurredAt    time.Time `gorm:"not null"`
	CreatedAt     time.Time
}
