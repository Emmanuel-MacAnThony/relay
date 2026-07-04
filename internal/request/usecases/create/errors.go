package create

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrSaveFailed        = errors.New("failed to save request")
	ErrIDConflict        = errors.New("id conflict")
	ErrEndpointNotFound  = errors.New("endpoint not found")
)
