package replay

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("request not found")
	ErrReplayFailed = errors.New("failed to replay request")
)
