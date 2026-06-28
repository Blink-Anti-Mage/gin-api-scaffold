package repository

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

	"github.com/example/gin-api-scaffold/app/example/service"
	"github.com/example/gin-api-scaffold/app/example/types"
	"github.com/example/gin-api-scaffold/internal/apperr"
)

var _ service.UsersRepository = (*PostgresUsersRepository)(nil)

type PostgresUsersRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUsersRepository(pool *pgxpool.Pool) *PostgresUsersRepository {
	return &PostgresUsersRepository{
		pool: pool,
	}
}

func (r *PostgresUsersRepository) List(ctx context.Context, filter types.ListUsersFilter) (types.UserList, error) {
	search := strings.TrimSpace(filter.Search)

	var total int64
	if err := r.pool.QueryRow(ctx, `
SELECT count(*)
FROM users
WHERE NULLIF($1, '') IS NULL
   OR name ILIKE '%' || $1 || '%'
   OR email ILIKE '%' || $1 || '%'`, search).Scan(&total); err != nil {
		return types.UserList{}, err
	}

	rows, err := r.pool.Query(ctx, `
SELECT id, name, email, created_at
FROM users
WHERE NULLIF($1, '') IS NULL
   OR name ILIKE '%' || $1 || '%'
   OR email ILIKE '%' || $1 || '%'
ORDER BY created_at ASC, id ASC
LIMIT $2 OFFSET $3`, search, filter.Limit, filter.Offset)
	if err != nil {
		return types.UserList{}, err
	}
	defer rows.Close()

	users := make([]types.User, 0)
	for rows.Next() {
		var user types.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt); err != nil {
			return types.UserList{}, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return types.UserList{}, err
	}

	return types.UserList{
		Items:  users,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (r *PostgresUsersRepository) Get(ctx context.Context, id string) (types.User, error) {
	var user types.User
	err := r.pool.QueryRow(ctx, `
SELECT id, name, email, created_at
FROM users
WHERE id = $1`, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return types.User{}, mapUserPostgresError(err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) Create(ctx context.Context, user types.User) (types.User, error) {
	if user.ID == "" {
		user.ID = newUserID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	err := r.pool.QueryRow(ctx, `
INSERT INTO users (id, name, email, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, created_at`,
		user.ID,
		user.Name,
		user.Email,
		user.CreatedAt,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return types.User{}, mapUserPostgresError(err)
	}

	return user, nil
}

func (r *PostgresUsersRepository) Stats(ctx context.Context) (types.UserStats, error) {
	var stats types.UserStats
	var lastCreatedAt pgtype.Timestamptz
	if err := r.pool.QueryRow(ctx, `
SELECT count(*), max(created_at)
FROM users`).Scan(&stats.Total, &lastCreatedAt); err != nil {
		return types.UserStats{}, err
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
