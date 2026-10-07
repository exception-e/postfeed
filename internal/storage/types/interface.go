package types

import (
	"context"
	"postfeed/internal/domain"

	"github.com/google/uuid"
)

//go:generate	mockgen -source=interface.go -destination=../../../internal/services/mocks/mockStorage.go -package=mocks

type PostRepo interface {
	CreatePost(ctx context.Context, post domain.Post) (uuid.UUID, error)
	UpdatePost(ctx context.Context, input PostUpdateInput) error
	ListPosts(ctx context.Context, cursor domain.Cursor, input PostListInput) ([]domain.Post, domain.Cursor, error)
}

type CommentRepo interface {
	CreateComment(ctx context.Context, comment domain.Comment) (uuid.UUID, error)
	ListComments(ctx context.Context, cursor domain.Cursor, input CommentListInput) ([]domain.Comment, domain.Cursor, error)
	ListFirstComments(ctx context.Context, first int64, postIDs []uuid.UUID) ([]domain.Comment, error)
	ListFirstReplies(ctx context.Context, first int64, parentIDs []uuid.UUID) ([]domain.Comment, error)
}
