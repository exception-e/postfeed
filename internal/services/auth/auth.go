package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
)

const userIDKey contextKey = "user-id"
const userIDHeader = "user-id"

type contextKey string

func AuthMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		errorResponse := func(msg string, args ...any) {
			err := graphql.ErrorResponse(r.Context(), msg, args...)

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)

			_ = json.NewEncoder(w).Encode(err)
		}

		userIDRaw := r.Header.Get(userIDHeader)
		if strings.TrimSpace(userIDRaw) == "" {
			errorResponse("UserID must be provided")
			return
		}

		userID, err := uuid.Parse(userIDRaw)
		if err != nil {
			errorResponse("UserID must be UUID")
			return
		}

		ctx = context.WithValue(ctx, userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func UserIDFromCtx(ctx context.Context) uuid.UUID {
	userIDRaw := ctx.Value(userIDKey)
	userID, _ := userIDRaw.(uuid.UUID)

	return userID
}
