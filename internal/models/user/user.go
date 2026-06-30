package user

import "time"

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type CreateUserInput struct {
	Name     string
	Email    string
	Password string
}

type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserInput struct {
	ID    string
	Name  string
	Email string
}

type ListUsersFilter struct {
	Search string
	Limit  int
	Cursor string
	After  *ListUsersCursor
}

type ListUsersCursor struct {
	CreatedAt time.Time
	ID        string
}

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserList struct {
	Items      []User `json:"items"`
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type UserStats struct {
	Total         int64      `json:"total"`
	LastCreatedAt *time.Time `json:"last_created_at,omitempty"`
}
