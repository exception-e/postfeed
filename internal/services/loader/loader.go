package loader

import (
	"context"
	"net/http"
	"postfeed/internal/domain"
	storageTypes "postfeed/internal/storage/types"
	"time"

	"github.com/google/uuid"
	"github.com/vikstrous/dataloadgen"
)

type ctxKey string

const (
	loadersKey = ctxKey("dataloaders")
)

// DataLoaders wrap your data loaders to inject via middleware.
type DataLoaders struct {
	FirstComments *dataloadgen.Loader[uuid.UUID, []*domain.Comment]
	FirstReplies  *dataloadgen.Loader[uuid.UUID, []*domain.Comment]
}

// NewDataLoaders .
func NewDataLoaders(commentsRepo storageTypes.CommentRepo) *DataLoaders {
	commentsFetchFn := func(ctx context.Context, postIDs []uuid.UUID) ([][]*domain.Comment, []error) {
		results := make([][]*domain.Comment, len(postIDs))
		errs := make([]error, len(postIDs))

		comments, err := commentsRepo.ListFirstComments(ctx, 30, postIDs)
		if err != nil {
			for i := 0; i < len(errs); i++ {
				errs[i] = err
			}
			return nil, errs
		}

		postIdxByID := make(map[uuid.UUID]int, len(postIDs))
		for idx, key := range postIDs {
			postIdxByID[key] = idx
		}
		for _, comment := range comments {
			resultsIdx := postIdxByID[comment.PostID]
			results[resultsIdx] = append(results[resultsIdx], &comment)
		}

		return results, errs
	}

	repliesFetchFn := func(ctx context.Context, parentIDs []uuid.UUID) ([][]*domain.Comment, []error) {
		results := make([][]*domain.Comment, len(parentIDs))
		errs := make([]error, len(parentIDs))

		comments, err := commentsRepo.ListFirstReplies(ctx, 30, parentIDs)
		if err != nil {
			for i := 0; i < len(errs); i++ {
				errs[i] = err
			}
			return nil, errs
		}

		parentIdxByID := make(map[uuid.UUID]int, len(parentIDs))
		for idx, key := range parentIDs {
			parentIdxByID[key] = idx
		}
		for _, comment := range comments {
			resultsIdx := parentIdxByID[comment.ParentID.UUID]
			results[resultsIdx] = append(results[resultsIdx], &comment)
		}

		return results, errs
	}

	return &DataLoaders{
		FirstComments: dataloadgen.NewLoader(commentsFetchFn, dataloadgen.WithWait(50*time.Millisecond)),
		FirstReplies:  dataloadgen.NewLoader(repliesFetchFn, dataloadgen.WithWait(50*time.Millisecond)),
	}
}

// Middleware injects data loaders into the context
func Middleware(commentsRepo storageTypes.CommentRepo, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loader := NewDataLoaders(commentsRepo)
		r = r.WithContext(context.WithValue(r.Context(), loadersKey, loader))
		next.ServeHTTP(w, r)
	})
}

// For returns the dataloader for a given context
func For(ctx context.Context) *DataLoaders {
	return ctx.Value(loadersKey).(*DataLoaders)
}
