package dto

type PostCreateRequest struct {
	Title string `json:"title" validate:"required,max=150"`
	Text  string `json:"text" validate:"omitempty"`
}

type PostUpdateRequest struct {
	Title *string `json:"title" validate:"omitempty,max=150"`
	Text  *string `json:"text" validate:"omitempty"`
}

type PostResponse struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Slug      string `json:"slug"`
	UserID    uint   `json:"user_id"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type PostResponseWithAuthor struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Slug      string `json:"slug"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Author    struct {
		ID       uint   `json:"id"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
	} `json:"author"`
}
