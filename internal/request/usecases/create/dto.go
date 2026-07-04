package create

import "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"

type CreateInput struct {
	Slug        string
	Method      string
	Path        string
	Headers     map[string][]string
	QueryParams map[string][]string
	Body        []byte
	SourceIP    string
}

type CreateOutput struct {
	Request domain.Request
}
