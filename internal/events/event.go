package events

import (
	"encoding/json"
	"go_blog/internal/models"
	"go_blog/internal/utils"
	"time"

	"github.com/google/uuid"
)

func NewPostCreatedEvent(post *models.Post) (*models.OutboxEvent, error) {
	payload := PostCreatedPayload{
		PostID: utils.UintToString(post.ID),
		Title:  post.Title,
		Slug:   post.Slug,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &models.OutboxEvent{
		EventID:       uuid.NewString(),
		Topic:         "blog.events", // Лучше вынести в константу
		EventType:     "PostCreated",
		AggregateType: "post",
		AggregateID:   utils.UintToString(post.ID),
		ActorUserID:   utils.UintToString(post.UserID),
		Payload:       string(payloadBytes), // В базе у тебя string, если переделаем на JSONB, уберем string()
		OccurredAt:    time.Now().UTC(),
		Status:        models.OutboxNew,
	}, nil
}

type Envelope struct {
	EventID    string    `json:"event_id"`   //UUID
	EventType  string    `json:"event_type"` //PostCreated,...
	OccurredAt time.Time `json:"occurred_at"`

	// для маршрутизации/упорядочивания
	AggregateType string `json:"aggregate_type"` //"post"
	AggregateID   string `json:"aggregate_id"`   //postID

	// полезно для трассировки
	ActorUserID   string `json:"actor_user_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`

	Version int             `json:"version"`
	Payload json.RawMessage `json:"payload"`
}
