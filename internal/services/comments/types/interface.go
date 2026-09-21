package types

import (
	"context"
	"postfeed/internal/domain"

	"github.com/google/uuid"
)

type CommentsService interface {
	CreateComment(ctx context.Context, input CreateCommentInput) (uuid.UUID, error)
	ListComments(ctx context.Context, input ListCommentsInput) ([]domain.Comment, domain.Cursor, error)
	ListReplies(ctx context.Context, input ListRepliesInput) ([]domain.Comment, domain.Cursor, error)
	GetComment(ctx context.Context, id uuid.UUID) (*domain.Comment, error)
}
