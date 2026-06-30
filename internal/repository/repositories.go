package repository

type Repositories struct {
	Users *PostgresUsersRepository
	Auth  *PostgresUsersRepository
}
