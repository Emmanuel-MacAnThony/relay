package create

import "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"

type CreateInput struct{}

type CreateOutput struct {
	Endpoint domain.Endpoint
}
