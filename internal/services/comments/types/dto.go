package types

import (
	"github.com/google/uuid"
)

type CreateCommentInput struct {
	PostID   uuid.UUID
	ParentID uuid.NullUUID
	UserID   uuid.UUID
	Body     string
}

type ListCommentsInput struct {
	PostID     uuid.UUID
	CursorNext *uuid.UUID
	Limit      int64
}

type ListRepliesInput struct {
	ParentID   uuid.UUID
	CursorNext *uuid.UUID
	Limit      int64
}
