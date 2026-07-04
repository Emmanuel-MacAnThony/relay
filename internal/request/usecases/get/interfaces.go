package get

import "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"

type Repository interface {
	Get(id string) (domain.Request, error)
}
