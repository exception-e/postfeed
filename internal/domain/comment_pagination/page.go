package comment_pagination

type Page[T any] struct {
	Items       []T
	NextCursor  string
	HasNextPage bool
}
