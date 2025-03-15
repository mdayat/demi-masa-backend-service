package dtos

type RegisterRequest struct {
	Username string `json:"username" validate:"omitempty,min=2"`
	Email    string `json:"email" validate:"omitempty,email"`
	Password string `json:"password" validate:"omitempty,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"omitempty,email"`
	Password string `json:"password" validate:"omitempty,min=8"`
}

type AuthResponse struct {
	RefreshToken string       `json:"refresh_token"`
	AccessToken  string       `json:"access_token"`
	User         UserResponse `json:"user"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}
