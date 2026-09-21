package domain

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID uuid.UUID

	UserID          uuid.UUID
	Body            string
	CommentsEnabled bool

	CreatedAt time.Time
	UpdatedAt time.Time
}
