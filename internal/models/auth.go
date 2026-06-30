package models

import "time"

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginInput struct {
	Email    string
	Password string
}

type LogoutInput struct {
	JWTID     string
	ExpiresAt time.Time
}

type LoginResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int64     `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
	Subject     string    `json:"subject"`
	Roles       []string  `json:"roles"`
	Scopes      []string  `json:"scopes"`
}

type CurrentUserResponse struct {
	Subject   string    `json:"subject"`
	Email     string    `json:"email,omitempty"`
	Name      string    `json:"name,omitempty"`
	Roles     []string  `json:"roles"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthUser struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
}
