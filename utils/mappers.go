package utils

import (
	"go_blog/dto"
	"go_blog/models"
)

func PostToResp(p models.Post) dto.PostResponse {
	return dto.PostResponse{
		ID:        p.ID,
		Title:     p.Title,
		Text:      p.Text,
		Slug:      p.Slug,
		UserID:    p.UserID,
		IsActive:  p.IsActive,
		CreatedAt: p.CreatedAt.Format("02.01.2006 15:04"),
		UpdatedAt: p.UpdatedAt.Format("02.01.2006 15:04"),
	}
}
