package graph

import (
	"errors"

	commentsServiceTypes "postfeed/internal/services/comments/types"
	PostsServiceTypes "postfeed/internal/services/posts/types"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	CodeNotAuthorized    = "NOT_AUTHORIZED"
	CodeNotFound         = "NOT_FOUND"
	CodeCommentsDisabled = "COMMENTS_DISABLED"
	CodeInvalidInput     = "INVALID_INPUT"
	CodeInternal         = "INTERNAL_ERROR"
)

func toGQLError(err error) error {
	switch {
	case errors.Is(err, commentsServiceTypes.ErrUnauthorized):
		return &gqlerror.Error{
			Message:    "unauthorized",
			Extensions: map[string]any{"code": CodeNotAuthorized},
		}
	case errors.Is(err, commentsServiceTypes.ErrInvalidInput):
		return &gqlerror.Error{
			Message:    "input parameters not valid",
			Extensions: map[string]any{"code": CodeInvalidInput, "field": "body"},
		}
	case errors.Is(err, PostsServiceTypes.ErrInvalidInput):
		return &gqlerror.Error{
			Message:    "input parameters not valid",
			Extensions: map[string]any{"code": CodeInvalidInput, "field": "body"},
		}
	case errors.Is(err, commentsServiceTypes.ErrCommentsDisabled):
		return &gqlerror.Error{
			Message:    "body is too long",
			Extensions: map[string]any{"code": CodeCommentsDisabled},
		}
	case errors.Is(err, commentsServiceTypes.ErrCommentNotFound):
		return &gqlerror.Error{
			Message:    "body is too long",
			Extensions: map[string]any{"code": CodeNotFound},
		}
	case errors.Is(err, commentsServiceTypes.ErrPostNotFound):
		return &gqlerror.Error{
			Message:    "body is too long",
			Extensions: map[string]any{"code": CodeNotFound},
		}
	default:
		// внутреннюю ошибку логируем, наружу не отдаём
		return &gqlerror.Error{
			Message:    "internal error",
			Extensions: map[string]any{"code": CodeInternal},
		}
	}
}
