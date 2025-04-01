package repo

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"github.com/google/uuid"
	"newservice/internal/config"
)

type repository struct {
	pool *pgxpool.Pool
}

type Repository interface {
	// методы работы с пользователями
	CreateUser(ctx context.Context, user *User) (uuid.UUID, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetPassword(ctx context.Context, userID uuid.UUID) (string, error)

	// методы работы с токенами
	NewRefreshToken(ctx context.Context, params NewRefreshTokenParams) (int64, error)
	DeleteRefreshToken(ctx context.Context, params DeleteRefreshTokenParams) error
	GetRefreshToken(ctx context.Context, params GetRefreshTokenParams) ([]string, error)
	UpdateRefreshToken(ctx context.Context, params UpdateRefreshTokenParams) error
	NewAuthToken(ctx context.Context, params NewAuthTokenParams) error

	// метод для graceful shutdown
	Close() error
}

const (
	createUserQuery = `
		INSERT INTO users (username, password_hash, email, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id;
	`

	getUserByUsernameQuery = `
		SELECT id, username, password_hash, email, created_at, updated_at
		FROM users
		WHERE username = $1;
	`

	getPasswordQuery = `
		SELECT password_hash
		FROM users
		WHERE id = $1;
	`

	insertRefreshTokenQuery = `
		INSERT INTO auth_tokens (user_id, refresh_token, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id;
	`

	deleteRefreshTokenQuery = `
		DELETE FROM auth_tokens
		WHERE user_id = $1;
	`

	getRefreshTokenQuery = `
		SELECT refresh_token
		FROM auth_tokens
		WHERE user_id = $1;
	`

	updateRefreshTokenQuery = `
		UPDATE auth_tokens
		SET refresh_token = $1, updated_at = NOW()
		WHERE user_id = $2;
	`
)

func NewRepository(ctx context.Context, cfg config.PostgreSQL) (Repository, error) {
	connString := fmt.Sprintf(
		`user=%s password=%s host=%s port=%d dbname=%s sslmode=%s 
         pool_max_conns=%d pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s`,
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
		cfg.PoolMaxConns,
		cfg.PoolMaxConnLifetime.String(),
		cfg.PoolMaxConnIdleTime.String(),
	)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse PostgreSQL config")
	}

	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PostgreSQL connection pool")
	}

	return &repository{pool: pool}, nil
}

func (r *repository) CreateUser(ctx context.Context, user *User) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, createUserQuery, user.Username, user.HashedPassword, user.Email).Scan(&id)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "failed to insert user")
	}
	return id, nil
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.pool.QueryRow(ctx, getUserByUsernameQuery, username).Scan(
		&user.ID,
		&user.Username,
		&user.HashedPassword,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user by username")
	}
	return &user, nil
}

func (r *repository) GetPassword(ctx context.Context, userID uuid.UUID) (string, error) {
	var password string
	err := r.pool.QueryRow(ctx, getPasswordQuery, userID).Scan(&password)
	if err != nil {
		return "", errors.Wrap(err, "failed to get password")
	}
	return password, nil
}

func (r *repository) NewRefreshToken(ctx context.Context, params NewRefreshTokenParams) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, insertRefreshTokenQuery, params.UserID, params.Token).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert refresh token")
	}
	return id, nil
}

func (r *repository) DeleteRefreshToken(ctx context.Context, params DeleteRefreshTokenParams) error {
	_, err := r.pool.Exec(ctx, deleteRefreshTokenQuery, params.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to delete refresh token")
	}
	return nil
}

func (r *repository) GetRefreshToken(ctx context.Context, params GetRefreshTokenParams) ([]string, error) {
	rows, err := r.pool.Query(ctx, getRefreshTokenQuery, params.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get refresh token")
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, errors.Wrap(err, "failed to scan refresh token")
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (r *repository) UpdateRefreshToken(ctx context.Context, params UpdateRefreshTokenParams) error {
	_, err := r.pool.Exec(ctx, updateRefreshTokenQuery, params.Token, params.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to update refresh token")
	}
	return nil
}

func (r *repository) NewAuthToken(ctx context.Context, params NewAuthTokenParams) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO auth_tokens (user_id, access_token, refresh_token, created_at, updated_at, access_expires_at, refresh_expires_at)
		VALUES ($1, $2, $3, NOW(), NOW(), NOW() + INTERVAL '1 hour', $4)`,
		params.UserID, params.Tokens.AccessToken, params.Tokens.RefreshToken, params.RefreshExpiresAt)
	if err != nil {
		return errors.Wrap(err, "failed to insert auth tokens")
	}
	return nil
}

// Close gracefully shuts down the database connection pool
func (r *repository) Close() error {
	if r.pool != nil {
		r.pool.Close()
	}
	return nil
}
