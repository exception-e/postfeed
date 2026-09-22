package pubsub

import (
	"context"
	"postfeed/internal/domain"
	"postfeed/internal/pubsub/types"

	"sync"

	"github.com/google/uuid"
)

const bufSize = 16 //TODO добавить в  .env

type PubSub struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan *domain.Comment]struct{}
	bufSize     int
}

func NewMemory() types.PubSub {
	return &PubSub{
		subscribers: make(map[uuid.UUID]map[chan *domain.Comment]struct{}),
		bufSize:     bufSize,
	}
}

func (ps *PubSub) Subscribe(ctx context.Context, postID uuid.UUID) (<-chan *domain.Comment, func()) {
	ch := make(chan *domain.Comment, ps.bufSize)

	ps.mu.Lock()
	if ps.subscribers[postID] == nil {
		ps.subscribers[postID] = make(map[chan *domain.Comment]struct{})
	}
	ps.subscribers[postID][ch] = struct{}{}
	ps.mu.Unlock()

	unsubscribe := func() {
		ps.mu.Lock()
		defer ps.mu.Unlock()

		subs, ok := ps.subscribers[postID]
		if !ok {
			return
		}
		if _, exists := subs[ch]; !exists {
			return
		}
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(ps.subscribers, postID)
		}
	}
	return ch, unsubscribe
}

func (ps *PubSub) Publish(ctx context.Context, postID uuid.UUID, comment *domain.Comment) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for ch := range ps.subscribers[postID] {
		select {
		case ch <- comment:
		default:
		}
	}
}
