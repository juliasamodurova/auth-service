package repo

import (
	"database/sql"
	"github.com/google/uuid"
	"newservice/pkg/jwt"
	"time"
)

type User struct {
	ID             uuid.UUID `db:"id"`
	Username       string    `db:"username"`
	HashedPassword string    `db:"password_hash"`
	Email          string    `db:"email"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type NewAuthTokenParams struct {
	UserID           uuid.UUID `db:"user_id"`
	Tokens           jwt.CreateTokenResponse
	RefreshExpiresAt time.Time `db:"refresh_expires_at"`
}

type NewRefreshTokenParams struct {
	UserID uuid.UUID `db:"user_id"`
	Token  string
}

type DeleteRefreshTokenParams struct {
	UserID uuid.UUID `db:"user_id"`
}

type GetRefreshTokenParams struct {
	UserID uuid.UUID `db:"user_id"`
}

type UpdateRefreshTokenParams struct {
	UserID    uuid.UUID `db:"user_id"`
	Token     string
	UpdatedAt sql.NullTime
}
