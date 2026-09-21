package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_ValidUUID(t *testing.T) {
	userID := uuid.New()

	var gotFromCtx uuid.UUID
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFromCtx = UserIDFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.Header.Set("user-id", userID.String())
	w := httptest.NewRecorder()

	AuthMiddleware(next).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, gotFromCtx, "middleware должен положить userID в контекст")
}

func TestUserIDFromCtx(t *testing.T) {
	t.Run("returns uuid from context", func(t *testing.T) {
		userID := uuid.New()
		var got uuid.UUID
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = UserIDFromCtx(r.Context())
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("user-id", userID.String())
		AuthMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

		assert.Equal(t, userID, got)
	})

	t.Run("returns zero uuid when not set", func(t *testing.T) {
		got := UserIDFromCtx(context.Background())
		assert.Equal(t, uuid.Nil, got)
	})

	t.Run("returns zero uuid when wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKey("user-id"), "not-a-uuid")
		got := UserIDFromCtx(ctx)
		assert.Equal(t, uuid.Nil, got)
	})
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true // не должен быть вызван
	})

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	w := httptest.NewRecorder()

	AuthMiddleware(next).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, called, "next НЕ должен быть вызван при отсутствии заголовка")

	// Проверяем тело ответа — это GraphQL-ошибка, а не plain text
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp, "errors")
}

func TestAuthMiddleware_InvalidUUID(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"not a uuid", "not-a-uuid"},
		{"empty-ish", " "},
		{"partial uuid", "123e4567-e89b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			})

			req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			req.Header.Set("user-id", tt.value)
			w := httptest.NewRecorder()

			AuthMiddleware(next).ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.False(t, called)
		})
	}
}

func TestAuthMiddleware_ErrorResponseFormat(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	w := httptest.NewRecorder()

	AuthMiddleware(next).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var resp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "UserID must be provided", resp.Errors[0].Message)
}
