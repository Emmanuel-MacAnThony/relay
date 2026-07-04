package create

import (
	"errors"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
	"github.com/Emmanuel-MacAnThony/relay/pkg/slug"
)

const maxSlugRetries = 3

type UseCase struct {
	repo    Repository
	slugGen func() (string, error)
}

func New(repo Repository) *UseCase {
	return &UseCase{repo: repo, slugGen: slug.Generate}
}

func NewWithSlugGen(repo Repository, slugGen func() (string, error)) *UseCase {
	return &UseCase{repo: repo, slugGen: slugGen}
}

func (uc *UseCase) Execute(_ CreateInput) result.Result[CreateOutput] {
	endpoint, err := uc.saveWithRetry()
	if err != nil {
		return result.Fail[CreateOutput](err)
	}
	return result.Ok(CreateOutput{Endpoint: endpoint})
}

func (uc *UseCase) saveWithRetry() (domain.Endpoint, error) {
	for i := 0; i < maxSlugRetries; i++ {
		s, err := uc.slugGen()
		if err != nil {
			return domain.Endpoint{}, ErrSlugGenerationFailed
		}

		e := domain.Endpoint{
			Slug:      s,
			CreatedAt: time.Now().UTC(),
		}

		err = uc.repo.Save(e)
		if err == nil {
			return e, nil
		}
		if errors.Is(err, ErrSlugConflict) {
			continue
		}
		return domain.Endpoint{}, ErrCreateFailed
	}
	return domain.Endpoint{}, ErrSlugConflict
}
