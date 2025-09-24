package models

// UserLogin represents login request data
type Token struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
