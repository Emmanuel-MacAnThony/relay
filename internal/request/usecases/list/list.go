package list

import "github.com/Emmanuel-MacAnThony/relay/pkg/result"

type UseCase struct {
	repo Repository
}

func New(repo Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(input ListInput) result.Result[ListOutput] {
	if input.Slug == "" {
		return result.Fail[ListOutput](ErrInvalidInput)
	}

	requests, err := uc.repo.List(input.Slug)
	if err != nil {
		return result.Fail[ListOutput](ErrListFailed)
	}

	return result.Ok(ListOutput{Requests: requests})
}
