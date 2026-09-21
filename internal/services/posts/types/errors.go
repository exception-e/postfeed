package types

import "errors"

var ErrUnauthorized = errors.New("unauthorized operation")
var ErrPostNotFound = errors.New("post not found")
var ErrInvalidInput = errors.New("input parameters invalid")
