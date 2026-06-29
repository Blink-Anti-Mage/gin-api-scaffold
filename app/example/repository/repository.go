package repository

import "github.com/example/gin-api-scaffold/app/example/service"

type Repositories struct {
	Users service.UsersRepository
	Auth  service.AuthRepository
}
