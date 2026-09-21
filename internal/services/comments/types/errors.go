package types

import "errors"

var ErrUnauthorized = errors.New("unauthorized operation")
var ErrCommentsDisabled = errors.New("comments disabled")
var ErrPostNotFound = errors.New("post not found")
var ErrCommentNotFound = errors.New("comment not found")
var ErrInvalidInput = errors.New("input parameters invalid")
