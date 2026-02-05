package events

import (
	"encoding/json"
	"github.com/google/uuid"
	"go_blog/internal/models"
	"go_blog/internal/utils"
	"time"
)

func NewCommentCreatedEvent(comment *models.Comment, postAuthorID uint, postSlug string) (*models.OutboxEvent, error) {
	payload := map[string]string{
		"comment_id":     utils.UintToString(comment.ID),
		"post_id":        utils.UintToString(comment.PostID),
		"post_slug":      postSlug,                         // Чтобы в уведомлении сделать ссылку
		"post_author_id": utils.UintToString(postAuthorID), // КОМУ отправлять уведомление
		"comment_text":   comment.Text,
		"author_id":      utils.UintToString(comment.UserID), // КТО написал
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &models.OutboxEvent{
		EventID:       uuid.NewString(),
		Topic:         "blog.comments", //не забыть добавить в consumer
		EventType:     "CommentCreated",
		AggregateType: "comment",
		AggregateID:   utils.UintToString(comment.ID),
		ActorUserID:   utils.UintToString(comment.UserID),
		Payload:       string(payloadBytes),
		OccurredAt:    time.Now().UTC(),
		Status:        models.OutboxNew,
	}, nil
}
