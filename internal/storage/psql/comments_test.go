package psql

import (
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

func TestStorage_CreateComment(t *testing.T) {
	ctx := t.Context()
	s := NewStorage(testDB, slog.New(tests.NewTestHandler(t)))
	userID := uuid.New()

	t.Run("create comment success", func(t *testing.T) {
		postID, err := s.CreatePost(ctx, domain.Post{
			UserID:          userID,
			Body:            "mock body",
			CommentsEnabled: false,
		})

		comment := domain.Comment{
			PostID: postID,
			UserID: userID,
			Body:   "first!",
		}
		id, err := s.CreateComment(ctx, comment)

		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, id)

		got, _, err := s.ListComments(ctx, domain.Cursor{
			Limit: 1,
		}, types.CommentListInput{
			PostIDs:        []uuid.UUID{postID},
			IDsIn:          []uuid.UUID{id},
			WithoutParents: true,
		})

		assert.Len(t, got, 1)
		assert.Equal(t, postID, got[0].PostID)
		assert.Equal(t, uuid.NullUUID{}, got[0].ParentID)
		assert.Equal(t, userID, got[0].UserID)
		assert.Equal(t, "first!", got[0].Body)
	})

	t.Run("existing Post, without ParentID", func(t *testing.T) {
		post := domain.Post{
			UserID:          uuid.New(),
			Body:            "mock text",
			CommentsEnabled: false,
		}

		postID, err := s.CreatePost(ctx, post)
		require.NoError(t, err)

		_, err = s.CreateComment(ctx, domain.Comment{
			PostID: postID,
			Body:   "mock comment text",
		})
		require.NoError(t, err)
	})

	t.Run("non-existing Post", func(t *testing.T) {
		_, err := s.CreateComment(ctx, domain.Comment{
			PostID: uuid.New(),
			Body:   "mock comment text",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, types.ErrPostNotFound)
	})

	t.Run("existing Post non-existing Parent", func(t *testing.T) {
		post := domain.Post{
			UserID:          uuid.New(),
			Body:            "mock text",
			CommentsEnabled: false,
		}

		postID, err := s.CreatePost(ctx, post)
		require.NoError(t, err)

		_, err = s.CreateComment(ctx, domain.Comment{
			UserID:   userID,
			PostID:   postID,
			ParentID: uuid.NullUUID{UUID: uuid.New(), Valid: true},
			Body:     "mock comment text",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, types.ErrParentCommentNotFound)
	})
}
func TestStorage_CommentCursor(t *testing.T) {
	ctx := t.Context()
	s := NewStorage(testDB, slog.New(tests.NewTestHandler(t)))

	userID := uuid.New()
	postId, err := s.CreatePost(ctx, domain.Post{
		UserID:          userID,
		Body:            "Post body",
		CommentsEnabled: true,
	})
	require.NoError(t, err)

	comments := make([]domain.Comment, 0, 5)
	for i := 0; i < 5; i++ {
		comment := domain.Comment{
			PostID: postId,
			UserID: userID,
			Body:   "post" + strconv.Itoa(i),
		}
		id, err := s.CreateComment(ctx, comment)
		require.NoError(t, err)
		comment.ID = id
		comments = append(comments, comment)
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].ID.String() < comments[j].ID.String()
	})

	t.Run("first page without cursor", func(t *testing.T) {
		actualComments, actualCursor, err := s.ListComments(ctx, domain.Cursor{Limit: 2}, types.CommentListInput{}) //types.PostListInput{IDs: []uuid.UUID{postIDs[0], postIDs[1]}}

		require.NoError(t, err)
		require.Len(t, actualComments, 2)
		require.Equal(t, comments[0].ID, actualComments[0].ID)
		require.Equal(t, comments[1].ID, actualComments[1].ID)
		require.NotNil(t, actualCursor.Next)
		require.Equal(t, comments[1].ID, *actualCursor.Next)
	})
	t.Run("next page uses cursor.Next", func(t *testing.T) {

		nextID := comments[1].ID
		actualComments, actualCursor, err := s.ListComments(ctx, domain.Cursor{Limit: 2, Next: &nextID}, types.CommentListInput{}) //types.PostListInput{IDs: []uuid.UUID{postIDs[2], postIDs[3]}}

		require.NoError(t, err)
		require.Len(t, actualComments, 2)
		require.Equal(t, comments[2].ID, actualComments[0].ID)
		require.Equal(t, comments[3].ID, actualComments[1].ID)
		require.NotNil(t, actualCursor.Next)
		require.Equal(t, comments[3].ID, *actualCursor.Next)
	})
	t.Run("prev page uses cursor.Prev", func(t *testing.T) {
		prevID := comments[3].ID
		actualComments, actualCursor, err := s.ListComments(ctx,
			domain.Cursor{Limit: 2, Prev: &prevID}, types.CommentListInput{})
		require.NoError(t, err)
		require.Len(t, actualComments, 2)
		require.Equal(t, comments[1].ID, actualComments[0].ID)
		require.Equal(t, comments[2].ID, actualComments[1].ID)
		require.NotNil(t, actualCursor.Prev)
		require.Equal(t, comments[1].ID, *actualCursor.Prev)
	})

	t.Run("last page has no next", func(t *testing.T) {
		nextID := comments[3].ID
		actualComments, actualCursor, err := s.ListComments(ctx,
			domain.Cursor{Limit: 2, Next: &nextID},
			types.CommentListInput{})
		require.NoError(t, err)
		require.Len(t, actualComments, 1)
		require.Equal(t, comments[4].ID, actualComments[0].ID)
		require.Nil(t, actualCursor.Next)
	})

	t.Run("limit must be GT 0", func(t *testing.T) {
		_, _, err := s.ListComments(ctx,
			domain.Cursor{Limit: 0},
			types.CommentListInput{},
		)
		require.Error(t, err)
	})

	t.Run("prev and next cannot be set together", func(t *testing.T) {
		a, b := comments[0].ID, comments[2].ID
		_, _, err := s.ListComments(ctx,
			domain.Cursor{Limit: 2, Next: &a, Prev: &b},
			types.CommentListInput{},
		)
		require.Error(t, err)
	})
}
