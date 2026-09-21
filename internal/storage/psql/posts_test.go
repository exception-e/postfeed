package psql

import (
	"fmt"
	"log/slog"
	"postfeed/internal/domain"
	"postfeed/internal/storage/types"
	"postfeed/internal/tests"
	"sort"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_CreatePost(t *testing.T) {
	ctx := t.Context()
	s := NewStorage(testDB, slog.New(tests.NewTestHandler(t)))

	post := domain.Post{
		UserID:          uuid.New(),
		Body:            "mock text",
		CommentsEnabled: false,
	}

	postID, err := s.CreatePost(ctx, post)
	require.NoError(t, err)

	actualPosts, actualCursor, err := s.ListPosts(ctx,
		domain.Cursor{
			Limit: 10,
		}, types.PostListInput{},
	)
	require.NoError(t, err)

	assert.EqualValues(t, 10, actualCursor.Limit)
	assert.Nil(t, actualCursor.Next)
	assert.Nil(t, actualCursor.Prev)

	require.Len(t, actualPosts, 1)
	actualPost := actualPosts[0]
	assert.Equal(t, postID, actualPost.ID)
	assert.Equal(t, post.UserID, actualPost.UserID)
	assert.Equal(t, post.Body, actualPost.Body)
	assert.NotEmpty(t, actualPost.CreatedAt)
	assert.NotEmpty(t, actualPost.UpdatedAt)

	trueVal := true
	err = s.UpdatePost(ctx, types.PostUpdateInput{
		ID:              postID,
		CommentsEnabled: &trueVal,
	})
	require.NoError(t, err)

	actualPosts, actualCursor, err = s.ListPosts(ctx,
		domain.Cursor{
			Limit: 10,
		}, types.PostListInput{},
	)
	require.NoError(t, err)

	require.Len(t, actualPosts, 1)
	actualPost = actualPosts[0]
	assert.Equal(t, trueVal, actualPost.CommentsEnabled)
}

func TestUpdatePost_IDRequired(t *testing.T) {
	ctx := t.Context()
	s := NewStorage(testDB, slog.New(tests.NewTestHandler(t)))
	err := s.UpdatePost(ctx, types.PostUpdateInput{ID: uuid.Nil})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestUpdatePost_CommentsEnabledFalse(t *testing.T) {
	ctx := t.Context()
	s := NewStorage(testDB, slog.New(tests.NewTestHandler(t)))
	post := domain.Post{
		UserID:          uuid.New(),
		Body:            "mock text",
		CommentsEnabled: false,
	}
	id, err := s.CreatePost(ctx, post)
	if err != nil {
		fmt.Errorf("inserting post: %w", err)
	}
	enabled := false

	err = s.UpdatePost(ctx, types.PostUpdateInput{
		ID:              id,
		CommentsEnabled: &enabled,
	})
	require.NoError(t, err)

	got, _, _ := s.ListPosts(ctx, domain.Cursor{
		Limit: 1,
	}, types.PostListInput{
		IDs: []uuid.UUID{id},
	})
	assert.Len(t, got, 1)
	assert.False(t, got[0].CommentsEnabled)
}

func TestUpdatePost_NotFound(t *testing.T) {
	ctx := t.Context()
	s := NewStorage(testDB, slog.New(tests.NewTestHandler(t)))
	err := s.UpdatePost(ctx, types.PostUpdateInput{ID: uuid.New()})

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrPostNotFound)
}

func TestStorage_PostCursor(t *testing.T) {
	ctx := t.Context()
	s := NewStorage(testDB, slog.New(tests.NewTestHandler(t)))

	userID := uuid.New()

	posts := make([]domain.Post, 0, 5)
	for i := 0; i < 5; i++ {
		p := domain.Post{
			UserID: userID,
			Body:   "post" + strconv.Itoa(i),
		}
		id, err := s.CreatePost(ctx, p)
		require.NoError(t, err)
		p.ID = id
		posts = append(posts, p)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].ID.String() < posts[j].ID.String()
	})

	t.Run("first page without cursor", func(t *testing.T) {
		actualPosts, actualCursor, err := s.ListPosts(ctx, domain.Cursor{Limit: 2}, types.PostListInput{}) //types.PostListInput{IDs: []uuid.UUID{postIDs[0], postIDs[1]}}

		require.NoError(t, err)
		require.Len(t, actualPosts, 2)
		require.Equal(t, posts[0].ID, actualPosts[0].ID)
		require.Equal(t, posts[1].ID, actualPosts[1].ID)
		require.NotNil(t, actualCursor.Next)
		require.Equal(t, posts[1].ID, *actualCursor.Next)
	})
	t.Run("next page uses cursor.Next", func(t *testing.T) {

		nextID := posts[1].ID
		actualPosts, actualCursor, err := s.ListPosts(ctx, domain.Cursor{Limit: 2, Next: &nextID}, types.PostListInput{}) //types.PostListInput{IDs: []uuid.UUID{postIDs[2], postIDs[3]}}

		require.NoError(t, err)
		require.Len(t, actualPosts, 2)
		require.Equal(t, posts[2].ID, actualPosts[0].ID)
		require.Equal(t, posts[3].ID, actualPosts[1].ID)
		require.NotNil(t, actualCursor.Next)
		require.Equal(t, posts[3].ID, *actualCursor.Next)
	})
	t.Run("prev page uses cursor.Prev", func(t *testing.T) {
		prevID := posts[3].ID
		actualPosts, actualCursor, err := s.ListPosts(ctx,
			domain.Cursor{Limit: 2, Prev: &prevID}, types.PostListInput{},
		)
		require.NoError(t, err)
		require.Len(t, actualPosts, 2)
		require.Equal(t, posts[1].ID, actualPosts[0].ID)
		require.Equal(t, posts[2].ID, actualPosts[1].ID)
		require.NotNil(t, actualCursor.Prev)
		require.Equal(t, posts[1].ID, *actualCursor.Prev)
	})

	t.Run("last page has no next", func(t *testing.T) {
		nextID := posts[3].ID
		actualPosts, actualCursor, err := s.ListPosts(ctx,
			domain.Cursor{Limit: 2, Next: &nextID},
			types.PostListInput{})
		require.NoError(t, err)
		require.Len(t, actualPosts, 1)
		require.Equal(t, posts[4].ID, actualPosts[0].ID)
		require.Nil(t, actualCursor.Next)
	})

	t.Run("limit must be GT 0", func(t *testing.T) {
		_, _, err := s.ListPosts(ctx,
			domain.Cursor{Limit: 0},
			types.PostListInput{},
		)
		require.Error(t, err)
	})

	t.Run("prev and next cannot be set together", func(t *testing.T) {
		a, b := posts[0].ID, posts[2].ID
		_, _, err := s.ListPosts(ctx,
			domain.Cursor{Limit: 2, Next: &a, Prev: &b},
			types.PostListInput{},
		)
		require.Error(t, err)
	})
}
