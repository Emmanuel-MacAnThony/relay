package list

import "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"

type ListInput struct {
	Slug string
}

type ListOutput struct {
	Requests []domain.Request
}
