package types

import (
	"context"
	"postfeed/internal/domain"

	"github.com/google/uuid"
)

type PostsService interface {
	CreatePost(ctx context.Context, input CreatePostInput) (uuid.UUID, error)
	SetCommentsEnabled(ctx context.Context, postID, userID uuid.UUID, enabled bool) error
	GetPost(ctx context.Context, id uuid.UUID) (*domain.Post, error)
	ListPosts(ctx context.Context, input ListPostsInput) ([]domain.Post, domain.Cursor, error)
}
