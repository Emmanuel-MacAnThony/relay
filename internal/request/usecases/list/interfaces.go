package list

import "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"

type Repository interface {
	List(slug string) ([]domain.Request, error)
}
