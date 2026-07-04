package list

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrListFailed   = errors.New("failed to list requests")
)
