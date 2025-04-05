package repo

import (
	"github.com/google/uuid"
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
