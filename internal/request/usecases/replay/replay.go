package replay

import (
	"errors"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
	"github.com/google/uuid"
)

type UseCase struct {
	repo     Repository
	forwarder Forwarder
	notifier  Notifier
}

func New(repo Repository, forwarder Forwarder, notifier Notifier) *UseCase {
	return &UseCase{repo: repo, forwarder: forwarder, notifier: notifier}
}

func (uc *UseCase) Execute(input ReplayInput) result.Result[ReplayOutput] {
	if input.ID == "" {
		return result.Fail[ReplayOutput](ErrInvalidInput)
	}

	req, err := uc.repo.Get(input.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return result.Fail[ReplayOutput](ErrNotFound)
		}
		return result.Fail[ReplayOutput](ErrReplayFailed)
	}

	go uc.asyncFlow(req)
	return result.Ok(ReplayOutput{})
}

func (uc *UseCase) asyncFlow(req domain.Request) {
	attempt, err := uc.forwarder.Forward(req)
	if err != nil {
		attempt = domain.DeliveryAttempt{
			ID:          uuid.New().String(),
			RequestID:   req.ID,
			Delivered:   false,
			IsReplay:    true,
			Error:       err.Error(),
			AttemptedAt: time.Now().UTC(),
		}
	} else {
		attempt.IsReplay = true
	}

	_ = uc.repo.SaveDeliveryAttempt(attempt)
	_ = uc.notifier.NotifyDelivery(attempt)
}
