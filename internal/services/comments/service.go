package comments

import (
	"context"
	"errors"
	"fmt"
	"postfeed/internal/domain"
	"postfeed/internal/services/comments/types"
	postsServiceTypes "postfeed/internal/services/posts/types"
	storageTypes "postfeed/internal/storage/types"
	"strings"

	"github.com/google/uuid"
)

var _ types.CommentsService = (*Service)(nil)

type Service struct {
	postsService postsServiceTypes.PostsService
	repo         storageTypes.CommentRepo
}

func NewService(postsService postsServiceTypes.PostsService, repo storageTypes.CommentRepo) *Service {
	return &Service{
		postsService: postsService,
		repo:         repo,
	}
}

func (s *Service) CreateComment(ctx context.Context, input types.CreateCommentInput) (uuid.UUID, error) {
	post, err := s.postsService.GetPost(ctx, input.PostID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("s.postsService.GetPost: %w", err)
	}
	if post == nil {
		return uuid.UUID{}, fmt.Errorf("s.postsService.GetPost: %w", types.ErrPostNotFound)
	}
	if !post.CommentsEnabled {
		return uuid.UUID{}, types.ErrCommentsDisabled
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return uuid.Nil, types.ErrInvalidInput
	}
	if len([]rune(body)) > 2000 { ///TODO добавить в env
		return uuid.Nil, types.ErrInvalidInput
	}

	comment := domain.Comment{
		PostID:   input.PostID,
		ParentID: input.ParentID,
		UserID:   input.UserID,
		Body:     input.Body,
	}

	id, err := s.repo.CreateComment(ctx, comment)
	if err != nil {
		switch {
		case errors.Is(err, storageTypes.ErrPostNotFound):
			return uuid.UUID{}, fmt.Errorf("s.repo.CreateComment: %w", types.ErrPostNotFound)
		case errors.Is(err, storageTypes.ErrCommentNotFound):
			return uuid.UUID{}, fmt.Errorf("s.repo.CreateComment: %w", types.ErrCommentNotFound)
		default:
			return uuid.UUID{}, fmt.Errorf("s.repo.CreateComment: %w", err)
		}
	}

	return id, nil
}

func (s *Service) ListComments(ctx context.Context, input types.ListCommentsInput) ([]domain.Comment, domain.Cursor, error) {
	cursor := domain.Cursor{
		Limit: input.Limit,
		Next:  input.CursorNext,
	}

	comments, nextCursor, err := s.repo.ListComments(ctx, cursor, storageTypes.CommentListInput{
		PostIDs:        []uuid.UUID{input.PostID},
		WithoutParents: true,
	})
	if err != nil {
		return nil, domain.Cursor{}, fmt.Errorf("s.repo.ListComments: %w", err)
	}

	return comments, nextCursor, nil
}

func (s *Service) GetComment(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	cursor := domain.Cursor{
		Limit: 1,
	}

	comments, _, err := s.repo.ListComments(ctx, cursor, storageTypes.CommentListInput{
		IDsIn: []uuid.UUID{id},
	})
	if err != nil {
		return nil, types.ErrCommentNotFound
	}
	if len(comments) == 0 {
		return nil, nil
	}

	return &comments[0], nil
}

func (s *Service) ListReplies(ctx context.Context, input types.ListRepliesInput) ([]domain.Comment, domain.Cursor, error) {
	cursor := domain.Cursor{
		Limit: input.Limit,
		Next:  input.CursorNext,
	}
	comments, nextCursor, err := s.repo.ListComments(ctx, cursor, storageTypes.CommentListInput{
		ParentIDsIn: []uuid.UUID{input.ParentID},
	})
	if err != nil {
		return nil, domain.Cursor{}, fmt.Errorf("s.repo.ListPosts: %w", err)
	}

	return comments, nextCursor, nil
}
