package inmemory

import (
	"bytes"
	"context"
	"fmt"
	"postfeed/internal/domain"
	"postfeed/internal/storage/types"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

var _ types.CommentRepo = (*CommentStorage)(nil)

type CommentStorage struct {
	mu       sync.RWMutex
	comments map[uuid.UUID]domain.Comment
	posts    *PostStorage // cсылка на PostStorage для эмуляции FK comments_post_id_fkey
}

func NewCommentStorage(posts *PostStorage) *CommentStorage {
	return &CommentStorage{
		comments: make(map[uuid.UUID]domain.Comment),
		posts:    posts,
	}
}

func (s *CommentStorage) CreateComment(ctx context.Context, comment domain.Comment) (uuid.UUID, error) {
	if s.posts != nil {
		s.posts.mu.RLock()
		_, ok := s.posts.posts[comment.PostID]
		s.posts.mu.RUnlock()
		if !ok {
			return uuid.UUID{}, fmt.Errorf("creating comment: %w", types.ErrPostNotFound)
		}
	}

	if comment.ParentID.Valid {
		s.mu.RLock()
		_, ok := s.comments[comment.ParentID.UUID]
		s.mu.RUnlock()
		if !ok {
			return uuid.UUID{}, fmt.Errorf("creating comment: %w", types.ErrParentCommentNotFound)
		}
	}

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if comment.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("generating uuidv7: %w", err)
		}
		comment.ID = id
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = now
	}
	s.comments[comment.ID] = comment

	return comment.ID, nil
}

func (s *CommentStorage) ListComments(ctx context.Context, cursor domain.Cursor, input types.CommentListInput) ([]domain.Comment, domain.Cursor, error) {
	if err := ctx.Err(); err != nil {
		return nil, domain.Cursor{}, err
	}
	if cursor.Limit <= 0 {
		return nil, domain.Cursor{}, fmt.Errorf("limit must be GT 0")
	}
	if cursor.Next != nil && cursor.Prev != nil {
		return nil, domain.Cursor{}, fmt.Errorf("prev / next cursor must be defined")
	}

	postIDsFilter := make(map[uuid.UUID]struct{}, len(input.PostIDs))
	for _, id := range input.PostIDs {
		postIDsFilter[id] = struct{}{}
	}
	parentIDsFilter := make(map[uuid.UUID]struct{}, len(input.ParentIDsIn))
	for _, id := range input.ParentIDsIn {
		parentIDsFilter[id] = struct{}{}
	}
	idsFilter := make(map[uuid.UUID]struct{}, len(input.IDsIn))
	for _, id := range input.IDsIn {
		idsFilter[id] = struct{}{}
	}

	s.mu.RLock()
	all := make([]domain.Comment, 0, len(s.comments))
	for _, c := range s.comments {
		if len(postIDsFilter) > 0 {
			if _, ok := postIDsFilter[c.PostID]; !ok {
				continue
			}
		}
		if input.WithoutParents && c.ParentID.Valid {
			continue
		}
		if len(parentIDsFilter) > 0 {
			if !c.ParentID.Valid {
				continue
			}
			if _, ok := parentIDsFilter[c.ParentID.UUID]; !ok {
				continue
			}
		}
		if len(idsFilter) > 0 {
			if _, ok := idsFilter[c.ID]; !ok {
				continue
			}
		}
		all = append(all, c)
	}
	s.mu.RUnlock()

	slices.SortFunc(all, func(a, b domain.Comment) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})

	var filtered []domain.Comment
	switch {
	case cursor.Next != nil:
		for _, c := range all {
			if bytes.Compare(c.ID[:], cursor.Next[:]) > 0 {
				filtered = append(filtered, c)
			}
		}
	case cursor.Prev != nil:
		for i := len(all) - 1; i >= 0; i-- {
			if bytes.Compare(all[i].ID[:], cursor.Prev[:]) < 0 {
				filtered = append(filtered, all[i])
			}
		}
	default:
		filtered = all
	}

	hasNext := int64(len(filtered)) > cursor.Limit
	if hasNext {
		filtered = filtered[:cursor.Limit]
	}
	if cursor.Prev != nil {
		slices.Reverse(filtered)
	}
	comments := filtered
	if comments == nil {
		comments = []domain.Comment{}
	}

	cursorNext := domain.Cursor{Limit: cursor.Limit}
	if len(comments) > 0 {
		firstID := comments[0].ID
		lastID := comments[len(comments)-1].ID

		switch {
		case cursor.Next == nil && cursor.Prev == nil:
			if hasNext {
				cursorNext.Next = &lastID
			}
		case cursor.Next != nil:
			cursorNext.Prev = &firstID
			if hasNext {
				cursorNext.Next = &lastID
			}
		case cursor.Prev != nil:
			cursorNext.Next = &lastID
			if hasNext {
				cursorNext.Prev = &firstID
			}
		}
	}

	return comments, cursorNext, nil
}

func (s *CommentStorage) ListFirstComments(ctx context.Context, first int64, postIDs []uuid.UUID) ([]domain.Comment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if first <= 0 {
		return nil, fmt.Errorf("first must be GT 0")
	}
	if len(postIDs) == 0 {
		return nil, fmt.Errorf("post ids must be provided")
	}

	postIDsFilter := make(map[uuid.UUID]struct{}, len(postIDs))
	for _, id := range postIDs {
		postIDsFilter[id] = struct{}{}
	}

	s.mu.RLock()
	all := make([]domain.Comment, 0, len(s.comments))
	for _, c := range s.comments {
		if _, ok := postIDsFilter[c.PostID]; !ok {
			continue
		}
		if c.ParentID.Valid {
			continue
		}
		all = append(all, c)
	}
	s.mu.RUnlock()

	slices.SortFunc(all, func(a, b domain.Comment) int {
		if cmp := bytes.Compare(a.PostID[:], b.PostID[:]); cmp != 0 {
			return cmp
		}
		return bytes.Compare(a.ID[:], b.ID[:])
	})

	comments := make([]domain.Comment, 0, first*int64(len(postIDs)))
	var (
		prevPostID uuid.UUID
		count      int64
		firstIter  = true
	)
	for _, c := range all {
		if firstIter || c.PostID != prevPostID {
			prevPostID = c.PostID
			count = 0
			firstIter = false
		}
		if count >= first {
			continue
		}
		comments = append(comments, c)
		count++
	}

	return comments, nil
}

func (s *CommentStorage) ListFirstReplies(ctx context.Context, first int64, parentIDs []uuid.UUID) ([]domain.Comment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if first <= 0 {
		return nil, fmt.Errorf("first must be GT 0")
	}
	if len(parentIDs) == 0 {
		return nil, fmt.Errorf("parent ids must be provided")
	}

	parentIDsFilter := make(map[uuid.UUID]struct{}, len(parentIDs))
	for _, id := range parentIDs {
		parentIDsFilter[id] = struct{}{}
	}

	s.mu.RLock()
	all := make([]domain.Comment, 0, len(s.comments))
	for _, c := range s.comments {
		if !c.ParentID.Valid {
			continue
		}
		if _, ok := parentIDsFilter[c.ParentID.UUID]; !ok {
			continue
		}
		all = append(all, c)
	}
	s.mu.RUnlock()

	slices.SortFunc(all, func(a, b domain.Comment) int {
		if cmp := bytes.Compare(a.ParentID.UUID[:], b.ParentID.UUID[:]); cmp != 0 {
			return cmp
		}
		return bytes.Compare(a.ID[:], b.ID[:])
	})

	comments := make([]domain.Comment, 0, first*int64(len(parentIDs)))
	var (
		prevParentID uuid.UUID
		count        int64
		firstIter    = true
	)
	for _, c := range all {
		if firstIter || c.ParentID.UUID != prevParentID {
			prevParentID = c.ParentID.UUID
			count = 0
			firstIter = false
		}
		if count >= first {
			continue
		}
		comments = append(comments, c)
		count++
	}
	return comments, nil
}
