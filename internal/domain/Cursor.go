package domain

import "github.com/google/uuid"

type Cursor struct {
	Limit int64
	Next  *uuid.UUID
	Prev  *uuid.UUID
}
