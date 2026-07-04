package get

import "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"

type GetInput struct {
	ID string
}

type GetOutput struct {
	Request domain.Request
}
