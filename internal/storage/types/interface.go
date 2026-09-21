package types

import (
	"context"
	"postfeed/internal/domain"

	"github.com/google/uuid"
)

type PostRepo interface {
	CreatePost(ctx context.Context, post domain.Post) (uuid.UUID, error)
	UpdatePost(ctx context.Context, input PostUpdateInput) error
	ListPosts(ctx context.Context, cursor domain.Cursor, input PostListInput) ([]domain.Post, domain.Cursor, error)
}

type CommentRepo interface {
	CreateComment(ctx context.Context, comment domain.Comment) (uuid.UUID, error)
	ListComments(ctx context.Context, cursor domain.Cursor, input CommentListInput) ([]domain.Comment, domain.Cursor, error)
}
