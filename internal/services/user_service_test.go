package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/example/gin-api-scaffold/internal/apperr"
	"github.com/example/gin-api-scaffold/internal/models"
)

type recordingUsersRepository struct {
	list   func(context.Context, models.ListUsersFilter) (models.UserList, error)
	create func(context.Context, models.User) (models.User, error)
	update func(context.Context, models.User) (models.User, error)
	delete func(context.Context, string) error
}

func (r *recordingUsersRepository) List(ctx context.Context, filter models.ListUsersFilter) (models.UserList, error) {
	if r.list == nil {
		return models.UserList{}, nil
	}
	return r.list(ctx, filter)
}

func (r *recordingUsersRepository) Get(context.Context, string) (models.User, error) {
	return models.User{}, nil
}

func (r *recordingUsersRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	if r.create == nil {
		return models.User{}, nil
	}
	return r.create(ctx, user)
}

func (r *recordingUsersRepository) Update(ctx context.Context, user models.User) (models.User, error) {
	if r.update == nil {
		return models.User{}, nil
	}
	return r.update(ctx, user)
}

func (r *recordingUsersRepository) Delete(ctx context.Context, id string) error {
	if r.delete == nil {
		return nil
	}
	return r.delete(ctx, id)
}

func (r *recordingUsersRepository) Stats(context.Context) (models.UserStats, error) {
	return models.UserStats{}, nil
}

func TestUsersServiceListUsesCursorPagination(t *testing.T) {
	firstCreatedAt := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	cursorUser := models.User{ID: "user-001", CreatedAt: firstCreatedAt}
	var captured models.ListUsersFilter

	repo := &recordingUsersRepository{
		list: func(_ context.Context, filter models.ListUsersFilter) (models.UserList, error) {
			captured = filter
			return models.UserList{
				Items: []models.User{
					{ID: "user-002", CreatedAt: firstCreatedAt.Add(time.Minute)},
					{ID: "user-003", CreatedAt: firstCreatedAt.Add(2 * time.Minute)},
					{ID: "user-004", CreatedAt: firstCreatedAt.Add(3 * time.Minute)},
				},
				Limit: filter.Limit,
			}, nil
		},
	}

	users, err := NewUsersService(repo).List(context.Background(), models.ListUsersFilter{
		Search: " ada ",
		Limit:  2,
		Cursor: encodeUsersListCursor(cursorUser),
	})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}

	if captured.Search != "ada" {
		t.Fatalf("expected trimmed search, got %q", captured.Search)
	}
	if captured.Limit != 3 {
		t.Fatalf("expected repository limit 3, got %d", captured.Limit)
	}
	if captured.After == nil {
		t.Fatal("expected decoded cursor")
	}
	if captured.After.ID != cursorUser.ID || !captured.After.CreatedAt.Equal(cursorUser.CreatedAt) {
		t.Fatalf("unexpected decoded cursor: %+v", captured.After)
	}

	if users.Limit != 2 {
		t.Fatalf("expected response limit 2, got %d", users.Limit)
	}
	if len(users.Items) != 2 {
		t.Fatalf("expected 2 returned users, got %d", len(users.Items))
	}
	if users.NextCursor == "" {
		t.Fatal("expected next cursor")
	}

	next, err := decodeUsersListCursor(users.NextCursor)
	if err != nil {
		t.Fatalf("decode next cursor: %v", err)
	}
	lastReturned := users.Items[len(users.Items)-1]
	if next.ID != lastReturned.ID || !next.CreatedAt.Equal(lastReturned.CreatedAt) {
		t.Fatalf("next cursor = %+v, want id %q created_at %s", next, lastReturned.ID, lastReturned.CreatedAt)
	}
}

func TestUsersServiceListRejectsInvalidCursor(t *testing.T) {
	repo := &recordingUsersRepository{
		list: func(context.Context, models.ListUsersFilter) (models.UserList, error) {
			t.Fatal("repository should not be called for invalid cursor")
			return models.UserList{}, nil
		},
	}

	_, err := NewUsersService(repo).List(context.Background(), models.ListUsersFilter{Cursor: "not-a-cursor"})
	if err == nil {
		t.Fatal("expected invalid cursor error")
	}

	appErr := apperr.From(err)
	if appErr.Code != "invalid_cursor" {
		t.Fatalf("expected invalid_cursor, got %q", appErr.Code)
	}
}

func TestUsersServiceCreateNormalizesInput(t *testing.T) {
	var captured models.User
	repo := &recordingUsersRepository{
		create: func(_ context.Context, user models.User) (models.User, error) {
			captured = user
			return user, nil
		},
	}

	user, err := NewUsersService(repo).Create(context.Background(), models.CreateUserInput{
		Name:     " Ada Byron ",
		Email:    " ADA@EXAMPLE.COM ",
		Password: "valid-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if captured.Name != "Ada Byron" {
		t.Fatalf("expected trimmed name, got %q", captured.Name)
	}
	if captured.Email != "ada@example.com" {
		t.Fatalf("expected normalized email, got %q", captured.Email)
	}
	if !passwordMatches(captured.PasswordHash, "valid-password") {
		t.Fatal("expected bcrypt password hash")
	}
	if user != captured {
		t.Fatalf("expected returned user to match repository result")
	}
}

func TestUsersServiceCreateValidatesInput(t *testing.T) {
	tests := []struct {
		name        string
		input       models.CreateUserInput
		expected    string
		expectedMsg string
	}{
		{
			name:        "empty name",
			input:       models.CreateUserInput{Name: "   ", Email: "ada@example.com", Password: "valid-password"},
			expected:    "invalid_name",
			expectedMsg: "name is required",
		},
		{
			name:        "name too long",
			input:       models.CreateUserInput{Name: strings.Repeat("a", maxUserNameLength+1), Email: "ada@example.com", Password: "valid-password"},
			expected:    "invalid_name",
			expectedMsg: "name too long",
		},
		{
			name:        "invalid email",
			input:       models.CreateUserInput{Name: "Ada Byron", Email: "not-an-email", Password: "valid-password"},
			expected:    "invalid_email",
			expectedMsg: "invalid email",
		},
		{
			name:        "password too short",
			input:       models.CreateUserInput{Name: "Ada Byron", Email: "ada@example.com", Password: "short"},
			expected:    "invalid_password",
			expectedMsg: "password too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &recordingUsersRepository{
				create: func(context.Context, models.User) (models.User, error) {
					t.Fatal("repository should not be called for invalid input")
					return models.User{}, nil
				},
			}

			_, err := NewUsersService(repo).Create(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected validation error")
			}

			appErr := apperr.From(err)
			if appErr.Code != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, appErr.Code)
			}
			if appErr.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, appErr.Message)
			}
		})
	}
}

func TestUsersServiceUpdateNormalizesInput(t *testing.T) {
	var captured models.User
	repo := &recordingUsersRepository{
		update: func(_ context.Context, user models.User) (models.User, error) {
			captured = user
			return user, nil
		},
	}

	user, err := NewUsersService(repo).Update(context.Background(), models.UpdateUserInput{
		ID:    " user-001 ",
		Name:  " Ada Byron ",
		Email: " ADA@EXAMPLE.COM ",
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}

	if captured.ID != "user-001" {
		t.Fatalf("expected trimmed id, got %q", captured.ID)
	}
	if captured.Name != "Ada Byron" {
		t.Fatalf("expected trimmed name, got %q", captured.Name)
	}
	if captured.Email != "ada@example.com" {
		t.Fatalf("expected normalized email, got %q", captured.Email)
	}
	if user != captured {
		t.Fatalf("expected returned user to match repository result")
	}
}

func TestUsersServiceDeleteTrimsID(t *testing.T) {
	var capturedID string
	repo := &recordingUsersRepository{
		delete: func(_ context.Context, id string) error {
			capturedID = id
			return nil
		},
	}

	if err := NewUsersService(repo).Delete(context.Background(), " user-001 "); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if capturedID != "user-001" {
		t.Fatalf("expected trimmed id, got %q", capturedID)
	}
}
