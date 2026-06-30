package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/gin-api-scaffold/internal/apperr"
	authmodel "github.com/example/gin-api-scaffold/internal/models/auth"
	usermodel "github.com/example/gin-api-scaffold/internal/models/user"
)

type PostgresUsersRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUsersRepository(pool *pgxpool.Pool) *PostgresUsersRepository {
	return &PostgresUsersRepository{
		pool: pool,
	}
}

func (r *PostgresUsersRepository) List(ctx context.Context, filter usermodel.ListUsersFilter) (usermodel.UserList, error) {
	search := strings.TrimSpace(filter.Search)

	var afterCreatedAt any
	var afterID string
	if filter.After != nil {
		afterCreatedAt = filter.After.CreatedAt
		afterID = filter.After.ID
	}

	rows, err := r.pool.Query(ctx, `
SELECT id, name, email, created_at
FROM users
WHERE (NULLIF($1, '') IS NULL
   OR name ILIKE '%' || $1 || '%'
   OR email ILIKE '%' || $1 || '%')
  AND ($2::timestamptz IS NULL OR (created_at, id) > ($2::timestamptz, $3::text))
ORDER BY created_at ASC, id ASC
LIMIT $4`, search, afterCreatedAt, afterID, filter.Limit)
	if err != nil {
		return usermodel.UserList{}, err
	}
	defer rows.Close()

	users := make([]usermodel.User, 0)
	for rows.Next() {
		var user usermodel.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt); err != nil {
			return usermodel.UserList{}, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return usermodel.UserList{}, err
	}

	return usermodel.UserList{
		Items: users,
		Limit: filter.Limit,
	}, nil
}

func (r *PostgresUsersRepository) Get(ctx context.Context, id string) (usermodel.User, error) {
	var user usermodel.User
	err := r.pool.QueryRow(ctx, `
SELECT id, name, email, created_at
FROM users
WHERE id = $1`, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return usermodel.User{}, mapUserPostgresError(err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) Create(ctx context.Context, user usermodel.User) (usermodel.User, error) {
	if user.ID == "" {
		user.ID = newUserID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	err := r.pool.QueryRow(ctx, `
INSERT INTO users (id, name, email, password_hash, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, email, created_at`,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return usermodel.User{}, mapUserPostgresError(err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) Update(ctx context.Context, user usermodel.User) (usermodel.User, error) {
	err := r.pool.QueryRow(ctx, `
UPDATE users
SET name = $2,
    email = $3
WHERE id = $1
RETURNING id, name, email, created_at`,
		user.ID,
		user.Name,
		user.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return usermodel.User{}, mapUserPostgresError(err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
DELETE FROM users
WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("user")
	}

	return nil
}

func (r *PostgresUsersRepository) GetByEmail(ctx context.Context, email string) (authmodel.AuthUser, error) {
	var user authmodel.AuthUser
	err := r.pool.QueryRow(ctx, `
SELECT id, name, email, password_hash
FROM users
WHERE lower(email) = lower($1)`, email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)
	if err != nil {
		return authmodel.AuthUser{}, mapUserPostgresError(err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) Stats(ctx context.Context) (usermodel.UserStats, error) {
	var stats usermodel.UserStats
	var lastCreatedAt pgtype.Timestamptz
	if err := r.pool.QueryRow(ctx, `
SELECT count(*), max(created_at)
FROM users`).Scan(&stats.Total, &lastCreatedAt); err != nil {
		return usermodel.UserStats{}, err
	}
	if lastCreatedAt.Valid {
		createdAt := lastCreatedAt.Time
		stats.LastCreatedAt = &createdAt
	}

	return stats, nil
}

func mapUserPostgresError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NotFound("user")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperr.Conflict("user_email_exists", "user email already exists")
	}

	return err
}

func newUserID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
