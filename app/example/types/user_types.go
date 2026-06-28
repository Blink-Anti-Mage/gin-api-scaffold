package types

import "time"

type CreateUserRequest struct {
	Name  string `json:"name" binding:"required,min=2,max=64"`
	Email string `json:"email" binding:"required,email,max=255"`
}

type CreateUserInput struct {
	Name  string
	Email string
}

type ListUsersFilter struct {
	Search string
	Limit  int
	Offset int
}

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserList struct {
	Items  []User `json:"items"`
	Total  int64  `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type UserStats struct {
	Total         int64      `json:"total"`
	LastCreatedAt *time.Time `json:"last_created_at,omitempty"`
}
