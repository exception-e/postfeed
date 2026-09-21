package comments

import (
	"context"
	"postfeed/internal/domain"
	"postfeed/internal/services/comments/types"
	"postfeed/internal/services/mocks"
	storageTypes "postfeed/internal/storage/types"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_CreateComment_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPosts := mocks.NewMockPostsService(ctrl)
	mockRepo := mocks.NewMockCommentRepo(ctrl)

	svc := NewService(mockPosts, mockRepo)

	postID := uuid.New()
	userID := uuid.New()
	commentID := uuid.New()
	ctx := context.Background()

	mockPosts.EXPECT().
		GetPost(gomock.Any(), postID).
		Return(&domain.Post{
			ID:              postID,
			CommentsEnabled: true,
		}, nil).
		Times(1)

	expectedComment := domain.Comment{
		PostID: postID,
		UserID: userID,
		Body:   "hello world",
	}
	mockRepo.EXPECT().
		CreateComment(gomock.Any(), expectedComment).
		Return(commentID, nil).
		Times(1)

	id, err := svc.CreateComment(ctx, types.CreateCommentInput{
		PostID: postID,
		UserID: userID,
		Body:   "hello world",
	})

	require.NoError(t, err)
	assert.Equal(t, commentID, id)
}
func TestService_CreateComment_PostNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPosts := mocks.NewMockPostsService(ctrl)
	mockRepo := mocks.NewMockCommentRepo(ctrl)
	svc := NewService(mockPosts, mockRepo)

	postID := uuid.New()
	ctx := context.Background()

	mockPosts.EXPECT().
		GetPost(gomock.Any(), postID).
		Return(nil, nil). // сервис вернул nil-пост без ошибки
		Times(1)

	_, err := svc.CreateComment(ctx, types.CreateCommentInput{
		PostID: postID,
		UserID: uuid.New(),
		Body:   "text",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrPostNotFound)
}

func TestService_CreateComment_CommentsDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPosts := mocks.NewMockPostsService(ctrl)
	mockRepo := mocks.NewMockCommentRepo(ctrl)
	svc := NewService(mockPosts, mockRepo)

	postID := uuid.New()

	mockPosts.EXPECT().
		GetPost(gomock.Any(), postID).
		Return(&domain.Post{ID: postID, CommentsEnabled: false}, nil)

	_, err := svc.CreateComment(context.Background(), types.CreateCommentInput{
		PostID: postID,
		UserID: uuid.New(),
		Body:   "text",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrCommentsDisabled)
}

func TestService_CreateComment_InvalidInput(t *testing.T) {
	postID := uuid.New()

	tests := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"spaces only", "    "},
		{"tabs and newlines", "\t\n  "},
		{"too long", string(make([]rune, 2001))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockPosts := mocks.NewMockPostsService(ctrl)
			mockRepo := mocks.NewMockCommentRepo(ctrl)
			svc := NewService(mockPosts, mockRepo)

			mockPosts.EXPECT().
				GetPost(gomock.Any(), postID).
				Return(&domain.Post{ID: postID, CommentsEnabled: true}, nil)

			_, err := svc.CreateComment(context.Background(), types.CreateCommentInput{
				PostID: postID,
				UserID: uuid.New(),
				Body:   tt.body,
			})

			require.Error(t, err)
			assert.ErrorIs(t, err, types.ErrInvalidInput)
		})
	}
}

func TestService_ListComments_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPosts := mocks.NewMockPostsService(ctrl)
	mockRepo := mocks.NewMockCommentRepo(ctrl)
	svc := NewService(mockPosts, mockRepo)

	postID := uuid.New()
	cursorNext := uuid.New()
	expNext := uuid.New()
	expectedComments := []domain.Comment{{ID: uuid.New(), PostID: postID}}
	expectedNext := domain.Cursor{Limit: 20, Next: &expNext}

	mockRepo.EXPECT().
		ListComments(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			cursor domain.Cursor,
			input storageTypes.CommentListInput,
		) ([]domain.Comment, domain.Cursor, error) {
			// Проверяем, что сервис сформировал корректный запрос
			assert.Equal(t, int64(20), cursor.Limit)
			assert.Equal(t, &cursorNext, cursor.Next)
			assert.Equal(t, []uuid.UUID{postID}, input.PostIDs)
			assert.True(t, input.WithoutParents)
			return expectedComments, expectedNext, nil
		})

	commentsRes, nextCursor, err := svc.ListComments(context.Background(), types.ListCommentsInput{
		PostID:     postID,
		Limit:      20,
		CursorNext: &cursorNext,
	})

	require.NoError(t, err)
	assert.Equal(t, expectedComments, commentsRes)
	assert.Equal(t, expectedNext, nextCursor)
}

func TestService_GetComment_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockPosts := mocks.NewMockPostsService(ctrl)
	mockRepo := mocks.NewMockCommentRepo(ctrl)
	svc := NewService(mockPosts, mockRepo)

	mockRepo.EXPECT().
		ListComments(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]domain.Comment{}, domain.Cursor{}, nil)

	got, err := svc.GetComment(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Nil(t, got)
}
