package api

type LoginRequest struct {
	Nickname string `json:"nickname" validate:"required,min=3,max=20"`
	Password string `json:"password" validate:"required,min=3"`
}

type RegisterRequest struct {
	Nickname string `json:"nickname" validate:"required,min=3,max=20"`
	Password string `json:"password" validate:"required,min=6"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}
