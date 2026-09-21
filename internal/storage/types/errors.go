package types

import "errors"

var ErrPostNotFound = errors.New("post not found")
var ErrCommentNotFound = errors.New("comment not found")
var ErrParentCommentNotFound = errors.New("parent comment not found")
