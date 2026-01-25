package events

type PostCreatedPayload struct {
	PostID string `json:"post_id"`
	Title  string `json:"title"`
	Slug   string `json:"slug"`
}

type PostUpdatedPayload struct {
	PostID string `json:"post_id"`
	Title  string `json:"title"`
	Slug   string `json:"slug"`
}

type PostDeletedPayload struct {
	PostID string `json:"post_id"`
}
