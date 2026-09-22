package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	pubsubTypes "postfeed/internal/pubsub/types"
	commentsServiceTypes "postfeed/internal/services/comments/types"
	postsServiceTypes "postfeed/internal/services/posts/types"
)

type Resolver struct {
	PostsService    postsServiceTypes.PostsService
	CommentsService commentsServiceTypes.CommentsService
	PubSub          pubsubTypes.PubSub
}
