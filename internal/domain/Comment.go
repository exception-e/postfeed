package domain

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	ParentID  uuid.NullUUID
	UserID    uuid.UUID
	Body      string
	CreatedAt time.Time
}
