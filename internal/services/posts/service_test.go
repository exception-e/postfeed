package posts

import (
	"context"
	"postfeed/internal/domain"
	"postfeed/internal/services/mocks"
	"postfeed/internal/services/posts/types"
	storageTypes "postfeed/internal/storage/types"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_CreatePost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRepo := mocks.NewMockPostRepo(ctrl)
	svc := NewService(mockRepo)

	userID := uuid.New()
	postID := uuid.New()

	mockRepo.EXPECT().
		CreatePost(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, post domain.Post) (uuid.UUID, error) {
			assert.Equal(t, userID, post.UserID)
			assert.Equal(t, "hello world", post.Body)
			assert.True(t, post.CommentsEnabled)
			return postID, nil
		})

	id, err := svc.CreatePost(context.Background(), types.CreatePostInput{
		UserID:          userID,
		Body:            "hello world",
		CommentsEnabled: true,
	})

	require.NoError(t, err)
	assert.Equal(t, postID, id)
}
func TestService_CreatePost_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"spaces only", "    "},
		{"tabs and newlines", "\t\n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockRepo := mocks.NewMockPostRepo(ctrl)
			svc := NewService(mockRepo)

			_, err := svc.CreatePost(context.Background(), types.CreatePostInput{
				UserID:          uuid.New(),
				Body:            tt.body,
				CommentsEnabled: true,
			})

			require.Error(t, err)
			assert.ErrorIs(t, err, types.ErrInvalidInput)
		})
	}
}

func TestService_GetPost_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRepo := mocks.NewMockPostRepo(ctrl)
	svc := NewService(mockRepo)

	mockRepo.EXPECT().
		ListPosts(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]domain.Post{}, domain.Cursor{}, nil)

	got, err := svc.GetPost(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Nil(t, got)
}
func TestService_SetCommentsEnabled_PostNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRepo := mocks.NewMockPostRepo(ctrl)
	svc := NewService(mockRepo)

	mockRepo.EXPECT().
		ListPosts(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]domain.Post{}, domain.Cursor{}, nil)

	// UpdatePost НЕ должен вызываться — EXPECT не настраиваем

	err := svc.SetCommentsEnabled(context.Background(), uuid.New(), uuid.New(), true)

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrPostNotFound)
}
func TestService_SetCommentsEnabled_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRepo := mocks.NewMockPostRepo(ctrl)
	svc := NewService(mockRepo)

	postID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	mockRepo.EXPECT().
		ListPosts(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]domain.Post{{ID: postID, UserID: ownerID}}, domain.Cursor{}, nil)

	// UpdatePost НЕ должен вызываться

	err := svc.SetCommentsEnabled(context.Background(), postID, otherUserID, true)

	require.Error(t, err)
	assert.ErrorIs(t, err, types.ErrUnauthorized)
}
func TestService_ListPosts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRepo := mocks.NewMockPostRepo(ctrl)
	svc := NewService(mockRepo)

	cursorNext := uuid.New()
	forNext := uuid.New()
	expectedPosts := []domain.Post{{ID: uuid.New()}}
	expectedNext := domain.Cursor{Limit: 20, Next: &forNext}

	mockRepo.EXPECT().
		ListPosts(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			cursor domain.Cursor,
			input storageTypes.PostListInput,
		) ([]domain.Post, domain.Cursor, error) {
			assert.Equal(t, int64(20), cursor.Limit)
			assert.Equal(t, &cursorNext, cursor.Next)
			// Сервис намеренно передаёт пустой фильтр — проверяем
			assert.Equal(t, storageTypes.PostListInput{}, input)
			return expectedPosts, expectedNext, nil
		})

	got, next, err := svc.ListPosts(context.Background(), types.ListPostsInput{
		Limit:      20,
		CursorNext: &cursorNext,
	})

	require.NoError(t, err)
	assert.Equal(t, expectedPosts, got)
	assert.Equal(t, expectedNext, next)
}
