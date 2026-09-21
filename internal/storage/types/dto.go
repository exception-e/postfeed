package types

import "github.com/google/uuid"

type PostUpdateInput struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CommentsEnabled *bool
}
type PostListInput struct {
	IDs []uuid.UUID
}

type CommentListInput struct {
	PostIDs []uuid.UUID
	IDsIn   []uuid.UUID
	//
	WithoutParents bool
	ParentIDsIn    []uuid.UUID
}
