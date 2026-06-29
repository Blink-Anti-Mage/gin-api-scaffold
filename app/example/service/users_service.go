package service

import (
	"context"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/example/gin-api-scaffold/app/example/types"
	"github.com/example/gin-api-scaffold/internal/apperr"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultUsersListLimit = 20
	MaxUsersListLimit     = 100
	maxUserNameLength     = 100
	maxUserEmailLength    = 255
	minUserPasswordLength = 8
	maxUserPasswordLength = 72
)

type UsersRepository interface {
	List(ctx context.Context, filter types.ListUsersFilter) (types.UserList, error)
	Get(ctx context.Context, id string) (types.User, error)
	Create(ctx context.Context, user types.User) (types.User, error)
	Update(ctx context.Context, user types.User) (types.User, error)
	Delete(ctx context.Context, id string) error
	Stats(ctx context.Context) (types.UserStats, error)
}

type UsersService struct {
	repo UsersRepository
}

func NewUsersService(repo UsersRepository) *UsersService {
	return &UsersService{
		repo: repo,
	}
}

func (s *UsersService) List(ctx context.Context, filter types.ListUsersFilter) (types.UserList, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	filter.Cursor = strings.TrimSpace(filter.Cursor)
	if filter.Limit <= 0 {
		filter.Limit = DefaultUsersListLimit
	}
	if filter.Limit > MaxUsersListLimit {
		filter.Limit = MaxUsersListLimit
	}

	limit := filter.Limit
	if filter.Cursor != "" {
		cursor, err := decodeUsersListCursor(filter.Cursor)
		if err != nil {
			return types.UserList{}, apperr.BadRequest("invalid_cursor", "cursor is invalid")
		}
		filter.After = cursor
	}

	filter.Limit = limit + 1
	users, err := s.repo.List(ctx, filter)
	if err != nil {
		return types.UserList{}, err
	}

	users.Limit = limit
	if len(users.Items) > limit {
		users.Items = users.Items[:limit]
		users.NextCursor = encodeUsersListCursor(users.Items[len(users.Items)-1])
	}

	return users, nil
}

func (s *UsersService) Get(ctx context.Context, id string) (types.User, error) {
	return s.repo.Get(ctx, id)
}

func (s *UsersService) Create(ctx context.Context, input types.CreateUserInput) (types.User, error) {
	user, err := normalizedUser(input.Name, input.Email)
	if err != nil {
		return types.User{}, err
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return types.User{}, err
	}
	user.PasswordHash = passwordHash

	return s.repo.Create(ctx, user)
}

func (s *UsersService) Update(ctx context.Context, input types.UpdateUserInput) (types.User, error) {
	user, err := normalizedUser(input.Name, input.Email)
	if err != nil {
		return types.User{}, err
	}
	user.ID = strings.TrimSpace(input.ID)

	return s.repo.Update(ctx, user)
}

func (s *UsersService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

func (s *UsersService) Stats(ctx context.Context) (types.UserStats, error) {
	return s.repo.Stats(ctx)
}

func normalizedUser(name string, email string) (types.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	if name == "" {
		return types.User{}, apperr.BadRequest("invalid_name", "name is required")
	}
	if utf8.RuneCountInString(name) > maxUserNameLength {
		return types.User{}, apperr.BadRequest("invalid_name", "name too long")
	}
	if !validEmail(email) {
		return types.User{}, apperr.BadRequest("invalid_email", "invalid email")
	}

	return types.User{
		Name:  name,
		Email: email,
	}, nil
}

func validEmail(email string) bool {
	if email == "" || len(email) > maxUserEmailLength || strings.ContainsAny(email, " \t\r\n") {
		return false
	}

	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func hashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", apperr.Internal(err)
	}
	return string(hash), nil
}

func passwordMatches(passwordHash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

func validatePassword(password string) error {
	if password == "" {
		return apperr.BadRequest("invalid_password", "password is required")
	}
	if len(password) < minUserPasswordLength {
		return apperr.BadRequest("invalid_password", "password too short")
	}
	if len(password) > maxUserPasswordLength {
		return apperr.BadRequest("invalid_password", "password too long")
	}
	return nil
}
