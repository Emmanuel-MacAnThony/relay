package create

import "errors"

var (
	ErrSlugGenerationFailed = errors.New("failed to generate slug")
	ErrCreateFailed         = errors.New("failed to create endpoint")
	ErrSlugConflict         = errors.New("slug already exists")
)
