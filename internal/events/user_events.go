package events

import (
	"encoding/json"
	"go_blog/internal/models"
	"go_blog/internal/utils"
	"time"

	"github.com/google/uuid"
)

// NewUserRegisteredEvent создает событие регистрации.
// (Пока Payload минимальный, потом расширим)
func NewUserRegisteredEvent(user *models.User) (*models.OutboxEvent, error) {
	payload := map[string]string{
		"user_id":  utils.UintToString(user.ID),
		"email":    user.Email,
		"nickname": user.Nickname,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &models.OutboxEvent{
		EventID:       uuid.NewString(),
		Topic:         "blog.users",
		EventType:     "UserRegistered",
		AggregateType: "user",
		AggregateID:   utils.UintToString(user.ID),
		ActorUserID:   utils.UintToString(user.ID), // Он сам себя создал
		Payload:       string(payloadBytes),
		OccurredAt:    time.Now().UTC(),
		Status:        models.OutboxNew,
	}, nil
}

func NewUserUpdatedEvent(user *models.User) (*models.OutboxEvent, error) {
	payload := map[string]string{
		"user_id":  utils.UintToString(user.ID),
		"email":    user.Email,
		"nickname": user.Nickname,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &models.OutboxEvent{
		EventID:       uuid.NewString(),
		Topic:         "blog.users",
		EventType:     "UserUpdated",
		AggregateType: "user",
		AggregateID:   utils.UintToString(user.ID),
		ActorUserID:   utils.UintToString(user.ID),
		Payload:       string(payloadBytes),
		OccurredAt:    time.Now().UTC(),
		Status:        models.OutboxNew,
	}, nil
}
