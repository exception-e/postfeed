package types

import "github.com/google/uuid"

type CreatePostInput struct {
	UserID          uuid.UUID
	Body            string
	CommentsEnabled bool
}

type ListPostsInput struct {
	CursorNext *uuid.UUID
	Limit      int64
}
