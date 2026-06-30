package services

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/example/gin-api-scaffold/internal/models"
)

type usersListCursorPayload struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeUsersListCursor(user models.User) string {
	payload := usersListCursorPayload{
		CreatedAt: user.CreatedAt.UTC(),
		ID:        user.ID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeUsersListCursor(raw string) (*models.ListUsersCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}

	var payload usersListCursorPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.ID == "" || payload.CreatedAt.IsZero() {
		return nil, errors.New("cursor is missing required fields")
	}

	return &models.ListUsersCursor{
		CreatedAt: payload.CreatedAt.UTC(),
		ID:        payload.ID,
	}, nil
}
