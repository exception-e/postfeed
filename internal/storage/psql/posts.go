package psql

import (
	"context"
	"fmt"
	"postfeed/internal/domain"
	"postfeed/internal/storage/types"
	"slices"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (s *Storage) CreatePost(ctx context.Context, post domain.Post) (uuid.UUID, error) {
	sqlQuery := `
	INSERT INTO posts (user_id, body, comments_enabled)
	VALUES ($1, $2, $3)
	RETURNING id
	`
	args := []any{post.UserID, post.Body, post.CommentsEnabled}

	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, sqlQuery, args...).
		Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("inserting post: %w", err)
	}

	return id, nil
}

func (s *Storage) UpdatePost(ctx context.Context, input types.PostUpdateInput) error {
	if input.ID == uuid.Nil {
		return fmt.Errorf("id is required")
	}

	args := []any{input.ID}

	sqlQuery := `
	UPDATE posts
	SET updated_at = NOW()
    `
	if input.CommentsEnabled != nil {
		sqlQuery += ",\ncomments_enabled = $2"
		args = append(args, *input.CommentsEnabled)
	}
	sqlQuery += "\nWHERE id = $1"

	result, err := s.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("updating post (id=%s): %w", input.ID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("updating post (id=%s): result.RowsAffected(): %w", input.ID, err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("updating post (id=%s): %w", input.ID, types.ErrPostNotFound)
	}

	return nil
}

func (s *Storage) ListPosts(ctx context.Context, cursor domain.Cursor, input types.PostListInput) ([]domain.Post, domain.Cursor, error) {
	if cursor.Limit <= 0 {
		return nil, domain.Cursor{}, fmt.Errorf("limit must be GT 0")
	}
	if cursor.Next != nil && cursor.Prev != nil {
		return nil, domain.Cursor{}, fmt.Errorf("prev / next cursor must be defined")
	}

	args := []any{cursor.Limit + 1}
	nextArgIdx := 2

	sqlQuery := `
	SELECT id, user_id, body, comments_enabled, created_at, updated_at
	FROM posts
	WHERE true
	`

	if len(input.IDs) > 0 {
		sqlQuery += fmt.Sprintf("\nAND id = ANY ($%d::uuid[])", nextArgIdx)
		args = append(args, pq.Array(input.IDs))
		nextArgIdx++
	}

	switch {
	case cursor.Next != nil:
		sqlQuery += fmt.Sprintf("\nAND id > ($%d::uuid)", nextArgIdx)
		args = append(args, *cursor.Next)
		nextArgIdx++
		sqlQuery += "\nORDER BY id ASC\nLIMIT $1"
	case cursor.Prev != nil:
		sqlQuery += fmt.Sprintf("\nAND id < ($%d::uuid)", nextArgIdx)
		args = append(args, *cursor.Prev)
		nextArgIdx++
		sqlQuery += "\nORDER BY id DESC\nLIMIT $1"
	default:
		sqlQuery += "\nORDER BY id ASC\nLIMIT $1"
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, domain.Cursor{}, fmt.Errorf("listing posts: %w", err)
	}
	defer rows.Close()

	posts := make([]domain.Post, 0, cursor.Limit+1)
	for rows.Next() {
		post := domain.Post{}
		err := rows.Scan(&post.ID, &post.UserID, &post.Body, &post.CommentsEnabled, &post.CreatedAt, &post.UpdatedAt)
		if err != nil {
			return nil, domain.Cursor{}, fmt.Errorf("listing posts: rows.Scan: %w", err)
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Cursor{}, fmt.Errorf("listing posts: rows.Err: %w", err)
	}

	hasNext := int64(len(posts)) > cursor.Limit
	if hasNext {
		posts = posts[:cursor.Limit]
	}

	if cursor.Prev != nil {
		slices.Reverse(posts)
	}

	cursorNext := domain.Cursor{Limit: cursor.Limit}
	if len(posts) > 0 {
		firstID := posts[0].ID
		lastID := posts[len(posts)-1].ID

		switch {
		case cursor.Next == nil && cursor.Prev == nil:
			if hasNext {
				cursorNext.Next = &lastID
			}
		case cursor.Next != nil:
			cursorNext.Prev = &firstID
			if hasNext {
				cursorNext.Next = &lastID
			}
		case cursor.Prev != nil:
			cursorNext.Next = &lastID
			if hasNext {
				cursorNext.Prev = &firstID
			}
		}
	}

	return posts, cursorNext, nil
}
