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

var _ types.PostRepo = (*PostStorage)(nil)

type PostStorage struct {
	mu    sync.RWMutex
	posts map[uuid.UUID]domain.Post
}

func NewPostStorage() *PostStorage {
	return &PostStorage{
		posts: make(map[uuid.UUID]domain.Post),
	}
}

func (s *PostStorage) CreatePost(ctx context.Context, post domain.Post) (uuid.UUID, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	if post.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("generating uuidv7: %w", err)
		}
		post.ID = id
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}

	post.UpdatedAt = now
	s.posts[post.ID] = post
	return post.ID, nil
}
func (s *PostStorage) UpdatePost(ctx context.Context, input types.PostUpdateInput) error {
	if input.ID == uuid.Nil {
		return fmt.Errorf("id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	post, ok := s.posts[input.ID]
	if !ok {
		return fmt.Errorf("updating post (id=%s): %w", input.ID, types.ErrPostNotFound)
	}
	post.CommentsEnabled = *input.CommentsEnabled
	post.UpdatedAt = time.Now().UTC()
	s.posts[input.ID] = post

	return nil
}

func (s *PostStorage) ListPosts(ctx context.Context, cursor domain.Cursor, input types.PostListInput) ([]domain.Post, domain.Cursor, error) {
	if cursor.Limit <= 0 {
		return nil, domain.Cursor{}, fmt.Errorf("limit must be GT 0")
	}
	if cursor.Next != nil && cursor.Prev != nil {
		return nil, domain.Cursor{}, fmt.Errorf("prev / next cursor must be defined")
	}

	idsFilter := make(map[uuid.UUID]struct{}, len(input.IDs))
	for _, id := range input.IDs {
		idsFilter[id] = struct{}{}
	}

	s.mu.RLock()
	all := make([]domain.Post, 0, len(s.posts))
	for _, post := range s.posts {
		if len(idsFilter) > 0 {
			if _, ok := idsFilter[post.ID]; !ok {
				continue
			}
		}
		all = append(all, post)
	}
	s.mu.RUnlock()

	slices.SortFunc(all, func(a, b domain.Post) int {
		return bytes.Compare(a.ID[:], b.ID[:])
	})

	var filtered []domain.Post
	switch {
	case cursor.Next != nil:
		for _, p := range all {
			if bytes.Compare(p.ID[:], cursor.Next[:]) > 0 {
				filtered = append(filtered, p)
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

	posts := filtered
	if posts == nil {
		posts = []domain.Post{}
	}

	cursorNext := domain.Cursor{Limit: cursor.Limit}
	if len(posts) > 0 {
		firstID := posts[0].ID
		lastID := posts[len(posts)-1].ID

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

	return posts, cursorNext, nil
}
