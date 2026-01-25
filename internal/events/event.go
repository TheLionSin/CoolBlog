package events

import (
	"encoding/json"
	"time"
)

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
