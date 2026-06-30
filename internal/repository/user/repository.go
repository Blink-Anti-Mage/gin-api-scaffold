package user

type Repositories struct {
	Users *PostgresUsersRepository
	Auth  *PostgresUsersRepository
}
