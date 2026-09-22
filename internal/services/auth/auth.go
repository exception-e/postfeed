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
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next.ServeHTTP(w, r)
			return
		}
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

		ctx = WithUserID(ctx, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func UserIDFromCtx(ctx context.Context) uuid.UUID {
	userIDRaw := ctx.Value(userIDKey)
	userID, _ := userIDRaw.(uuid.UUID)

	return userID
}
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// достаёт userID из connection_init payload WebSocket, ожидает payload вида {"user-id": "<uuid>"}
func ParseUserIDFromInitPayload(payload map[string]any) (uuid.UUID, bool) {
	raw, ok := payload["user-id"]
	if !ok {
		return uuid.Nil, false
	}
	s, ok := raw.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
