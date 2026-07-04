package get

import (
	"errors"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
)

type UseCase struct {
	repo Repository
}

func New(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(input GetInput) result.Result[GetOutput] {
	if input.ID == "" {
		return result.Fail[GetOutput](ErrInvalidInput)
	}

	req, err := uc.repo.Get(input.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return result.Fail[GetOutput](ErrNotFound)
		}
		return result.Fail[GetOutput](ErrGetFailed)
	}

	return result.Ok(GetOutput{Request: req})
}
