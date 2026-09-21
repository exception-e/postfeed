package posts

import (
	"context"
	"fmt"
	"postfeed/internal/domain"
	"postfeed/internal/services/posts/types"
	storageTypes "postfeed/internal/storage/types"
	"strings"

	"github.com/google/uuid"
)

//go:generate	mockgen -source=./internal/services/posts/types/interface.go -destination=./internal/services/mocks/mockPostsService.go -package=mocks

var _ types.PostsService = (*Service)(nil)

type Service struct {
	repo storageTypes.PostRepo
}

func NewService(repo storageTypes.PostRepo) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreatePost(ctx context.Context, input types.CreatePostInput) (uuid.UUID, error) {
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return uuid.Nil, types.ErrInvalidInput
	}

	post := domain.Post{
		UserID:          input.UserID,
		Body:            input.Body,
		CommentsEnabled: input.CommentsEnabled,
	}
	postID, err := s.repo.CreatePost(ctx, post)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("s.repo.CreatePost: %w", err)
	}

	return postID, nil
}

func (s *Service) SetCommentsEnabled(ctx context.Context, postID, userID uuid.UUID, enabled bool) error {
	post, err := s.GetPost(ctx, postID)
	if err != nil {
		return fmt.Errorf("s.GetPost: %w", err)
	}
	if post == nil {
		return fmt.Errorf("s.GetPost: %w", types.ErrPostNotFound)
	}
	if post.UserID != userID {
		return types.ErrUnauthorized
	}

	input := storageTypes.PostUpdateInput{
		ID:              postID,
		CommentsEnabled: &enabled,
	}
	err = s.repo.UpdatePost(ctx, input)
	if err != nil {
		return fmt.Errorf("s.repo.UpdatePost failed %w", err)
	}
	return nil
}

func (s *Service) ListPosts(ctx context.Context, input types.ListPostsInput) ([]domain.Post, domain.Cursor, error) {
	cursor := domain.Cursor{
		Limit: input.Limit,
		Next:  input.CursorNext,
	}

	posts, nextCursor, err := s.repo.ListPosts(ctx, cursor, storageTypes.PostListInput{})
	if err != nil {
		return nil, domain.Cursor{}, fmt.Errorf("s.repo.ListPosts: %w", err)
	}

	return posts, nextCursor, nil
}

func (s *Service) GetPost(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	posts, _, err := s.repo.ListPosts(
		ctx,
		domain.Cursor{
			Limit: 1,
		},
		storageTypes.PostListInput{
			IDs: []uuid.UUID{id},
		})
	if err != nil {
		return nil, fmt.Errorf("s.repo.ListPosts: %w", err)
	}
	if len(posts) == 0 {
		return nil, nil
	}

	return &posts[0], nil
}
