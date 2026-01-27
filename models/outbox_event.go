package models

import "time"

type OutboxStatus string

const (
	OutboxNew  OutboxStatus = "NEW"
	OutboxSent OutboxStatus = "SENT"
)

type OutboxEvent struct {
	ID            uint         `gorm:"primaryKey"`
	EventID       string       `gorm:"size:36;uniqueIndex;not null"`
	Topic         string       `gorm:"size:200;not null"`   // blog.events
	EventType     string       `gorm:"size:50;not null"`    // PostCreated
	AggregateType string       `gorm:"size:50;not null"`    // post
	AggregateID   string       `gorm:"size:50;not null"`    // postID
	ActorUserID   string       `gorm:"size:50"`             // кто сделал
	Payload       string       `gorm:"type:jsonb;not null"` // JSON строкой
	OccurredAt    time.Time    `gorm:"not null"`
	Status        OutboxStatus `gorm:"size:10;not null;index"` //NEW SENT
	Attempts      int          `gorm:"not null;default:0"`     //ПОПЫТКИ
	LastError     string       `gorm:"type:text"`
	SentAt        *time.Time
	CreatedAt     time.Time
}
