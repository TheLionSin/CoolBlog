package dto

type UserMeResponse struct {
	ID       uint   `json:"id"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type UserUpdateReq struct {
	Nickname *string `json:"nickname" validate:"omitempty,min=3,max=30"`
}
