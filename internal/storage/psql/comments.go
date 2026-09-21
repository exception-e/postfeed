package psql

import (
	"context"
	"errors"
	"fmt"
	"postfeed/internal/domain"
	"postfeed/internal/storage/types"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
)

var _ types.CommentRepo = (*Storage)(nil)

func (s *Storage) CreateComment(ctx context.Context, comment domain.Comment) (uuid.UUID, error) {
	sqlQuery := `
	INSERT INTO comments (post_id, parent_id, user_id, body)
	VALUES ($1, $2, $3, $4)
	RETURNING id
	`
	args := []any{comment.PostID, comment.ParentID, comment.UserID, comment.Body}

	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, sqlQuery, args...).
		Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.ConstraintName == "comments_post_id_fkey" {
				return uuid.UUID{}, fmt.Errorf("creating comment: %w", types.ErrPostNotFound)
			}
			if pgErr.ConstraintName == "comments_parent_id_fkey" {
				return uuid.UUID{}, fmt.Errorf("creating comment: %w", types.ErrParentCommentNotFound)
			}
		}

		return uuid.UUID{}, fmt.Errorf("inserting comment: %w", err)
	}

	return id, nil
}

func (s *Storage) ListComments(ctx context.Context, cursor domain.Cursor, input types.CommentListInput) ([]domain.Comment, domain.Cursor, error) {
	if cursor.Limit <= 0 {
		return nil, domain.Cursor{}, fmt.Errorf("limit must be GT 0")
	}
	if cursor.Next != nil && cursor.Prev != nil {
		return nil, domain.Cursor{}, fmt.Errorf("prev / next cursor must be defined")
	}

	args := []any{cursor.Limit + 1}
	nextArgIdx := 2

	sqlQuery := `
	SELECT id, user_id, post_id, parent_id, body, created_at
	FROM comments
	WHERE true
	`

	if len(input.PostIDs) > 0 {
		sqlQuery += fmt.Sprintf("\nAND post_id = ANY ($%d::uuid[])", nextArgIdx)
		args = append(args, pq.Array(input.PostIDs))
		nextArgIdx++
	}

	if input.WithoutParents {
		sqlQuery += "\nAND parent_id IS NULL"
	}

	if len(input.ParentIDsIn) > 0 {
		sqlQuery += fmt.Sprintf("\nAND parent_id = ANY ($%d::uuid[])", nextArgIdx)
		args = append(args, pq.Array(input.ParentIDsIn))
		nextArgIdx++
	}

	if len(input.IDsIn) > 0 {
		sqlQuery += fmt.Sprintf("\nAND id = ANY ($%d::uuid[])", nextArgIdx)
		args = append(args, pq.Array(input.IDsIn))
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
		return nil, domain.Cursor{}, fmt.Errorf("listing comments: %w", err)
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0, cursor.Limit+1)
	for rows.Next() {
		comment := domain.Comment{}
		err := rows.Scan(
			&comment.ID,
			&comment.UserID,
			&comment.PostID,
			&comment.ParentID,
			&comment.Body,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, domain.Cursor{}, fmt.Errorf("listing comments: rows.Scan: %w", err)
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Cursor{}, fmt.Errorf("listing comments: rows.Err: %w", err)
	}

	hasNext := int64(len(comments)) > cursor.Limit
	if hasNext {
		comments = comments[:cursor.Limit]
	}

	if cursor.Prev != nil {
		slices.Reverse(comments)
	}

	cursorNext := domain.Cursor{Limit: cursor.Limit}
	if len(comments) > 0 {
		firstID := comments[0].ID
		lastID := comments[len(comments)-1].ID

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
	return comments, cursorNext, nil
}

func (s *Storage) ListFirstComments(ctx context.Context, first int64, postIDs []uuid.UUID) ([]domain.Comment, error) {
	if first <= 0 {
		return nil, fmt.Errorf("first must be GT 0")
	}
	if len(postIDs) == 0 {
		return nil, fmt.Errorf("post ids must be provided")
	}

	args := []any{first, pq.Array(postIDs)}

	sqlQuery := `
	WITH numbered_comments AS (
    SELECT
        c.id,
        c.user_id,
        c.post_id,
        c.parent_id,
        c.body,
        c.created_at,
        row_number() OVER (
            PARTITION BY c.post_id
            ORDER BY c.id ASC
        ) AS rn
    FROM comments c
    WHERE c.post_id = ANY ($2::uuid[])
      AND c.parent_id IS NULL
	)
	SELECT
    	id,
    	user_id,
    	post_id,
    	parent_id,
    	body,
    	created_at
	FROM numbered_comments
	WHERE rn <= $1
	ORDER BY post_id, id;
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0, first)
	for rows.Next() {
		comment := domain.Comment{}
		err := rows.Scan(
			&comment.ID,
			&comment.UserID,
			&comment.PostID,
			&comment.ParentID,
			&comment.Body,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("listing comments: rows.Scan: %w", err)
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing comments: rows.Err: %w", err)
	}

	return comments, nil
}

func (s *Storage) ListFirstReplies(ctx context.Context, first int64, parentIDs []uuid.UUID) ([]domain.Comment, error) {
	if first <= 0 {
		return nil, fmt.Errorf("first must be GT 0")
	}
	if len(parentIDs) == 0 {
		return nil, fmt.Errorf("parent ids must be provided")
	}

	args := []any{first, pq.Array(parentIDs)}

	sqlQuery := `
	WITH numbered_comments AS (
    SELECT
        c.id,
        c.user_id,
        c.post_id,
        c.parent_id,
        c.body,
        c.created_at,
        row_number() OVER (
            PARTITION BY c.parent_id
            ORDER BY c.id ASC
        ) AS rn
    FROM comments c
    WHERE c.parent_id = ANY ($2::uuid[])
)
SELECT
    id,
    user_id,
    post_id,
    parent_id,
    body,
    created_at
FROM numbered_comments
WHERE rn <= $1
ORDER BY parent_id, id;
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("listing replies: %w", err)
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0, first)
	for rows.Next() {
		comment := domain.Comment{}
		err := rows.Scan(
			&comment.ID,
			&comment.UserID,
			&comment.PostID,
			&comment.ParentID,
			&comment.Body,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("listing replies: rows.Scan: %w", err)
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing replies: rows.Err: %w", err)
	}

	return comments, nil
}
