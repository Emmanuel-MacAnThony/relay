package get

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("request not found")
	ErrGetFailed    = errors.New("failed to get request")
)
