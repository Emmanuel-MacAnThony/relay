package domain

import (
	"fmt"
	"time"
)

type Endpoint struct {
	Slug      string
	CreatedAt time.Time
}

func (e Endpoint) URL(baseURL string) string {
	return fmt.Sprintf("%s/hook/%s", baseURL, e.Slug)
}
