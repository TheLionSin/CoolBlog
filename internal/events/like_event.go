package events

import (
	"encoding/json"
	"go_blog/internal/models"
	"go_blog/internal/utils"
	"time"

	"github.com/google/uuid"
)

func NewPostLikedEvent(like *models.PostLike, postAuthorID uint, postSlug string) (*models.OutboxEvent, error) {
	payload := map[string]string{
		"like_id":        utils.UintToString(like.ID),
		"post_id":        utils.UintToString(like.PostID),
		"post_slug":      postSlug,
		"post_author_id": utils.UintToString(postAuthorID), //Кому отправить уведомление
		"liker_id":       utils.UintToString(like.UserID),  //Кто лайкнул
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &models.OutboxEvent{
		EventID:       uuid.NewString(),
		Topic:         "blog.likes", // Новый топик
		EventType:     "PostLiked",
		AggregateType: "post_like",
		AggregateID:   utils.UintToString(like.PostID), // Агрегат по посту (или по лайку)
		ActorUserID:   utils.UintToString(like.UserID),
		Payload:       string(payloadBytes),
		OccurredAt:    time.Now().UTC(),
		Status:        models.OutboxNew,
	}, nil
}
