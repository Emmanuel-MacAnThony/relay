package proxy

import "errors"

var (
	ErrNoCLI           = errors.New("no CLI connected for slug")
	ErrCLITimeout      = errors.New("CLI did not respond in time")
	ErrAlreadyInFlight = errors.New("request already in flight")
	ErrMarshalFailed   = errors.New("failed to marshal request")
	ErrWriteFailed     = errors.New("failed to send request")
	ErrReadFailed      = errors.New("failed to read response")
	ErrInvalidResponse = errors.New("invalid response from CLI")
)
