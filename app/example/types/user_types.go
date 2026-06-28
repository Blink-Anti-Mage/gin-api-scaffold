package types

import "time"

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CreateUserInput struct {
	Name  string
	Email string
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
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
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
