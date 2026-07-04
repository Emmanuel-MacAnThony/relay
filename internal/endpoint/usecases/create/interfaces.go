package create

import "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"

type Repository interface {
	Save(e domain.Endpoint) error
}
