package service

import (
	"context"
	"strings"

	"github.com/example/gin-api-scaffold/app/example/types"
)

const (
	defaultUsersListLimit = 20
	maxUsersListLimit     = 100
)

type UsersRepository interface {
	List(ctx context.Context, filter types.ListUsersFilter) (types.UserList, error)
	Get(ctx context.Context, id string) (types.User, error)
	Create(ctx context.Context, user types.User) (types.User, error)
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
	if filter.Limit <= 0 {
		filter.Limit = defaultUsersListLimit
	}
	if filter.Limit > maxUsersListLimit {
		filter.Limit = maxUsersListLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.repo.List(ctx, filter)
}

func (s *UsersService) Get(ctx context.Context, id string) (types.User, error) {
	return s.repo.Get(ctx, id)
}

func (s *UsersService) Create(ctx context.Context, input types.CreateUserInput) (types.User, error) {
	return s.repo.Create(ctx, types.User{
		Name:  strings.TrimSpace(input.Name),
		Email: strings.ToLower(strings.TrimSpace(input.Email)),
	})
}

func (s *UsersService) Stats(ctx context.Context) (types.UserStats, error) {
	return s.repo.Stats(ctx)
}
