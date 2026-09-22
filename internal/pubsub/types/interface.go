package types

import (
	"context"
	"postfeed/internal/domain"

	"github.com/google/uuid"
)

type PubSub interface {
	Publish(ctx context.Context, postID uuid.UUID, comment *domain.Comment)
	Subscribe(ctx context.Context, postID uuid.UUID) (<-chan *domain.Comment, func())
}
