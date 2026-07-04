package create

import (
	"errors"
	"time"

	endpointdomain "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
	"github.com/google/uuid"
)

const maxIDRetries = 3

type UseCase struct {
	repo         Repository
	endpointRepo EndpointRepository
	forwarder    Forwarder
	notifier     Notifier
}

func New(repo Repository, endpointRepo EndpointRepository, forwarder Forwarder, notifier Notifier) *UseCase {
	return &UseCase{repo: repo, endpointRepo: endpointRepo, forwarder: forwarder, notifier: notifier}
}

func (uc *UseCase) Execute(input CreateInput) result.Result[CreateOutput] {
	if input.Slug == "" || input.Method == "" {
		return result.Fail[CreateOutput](ErrInvalidInput)
	}

	_, err := uc.endpointRepo.Get(input.Slug)
	if err != nil {
		if errors.Is(err, endpointdomain.ErrNotFound) {
			return result.Fail[CreateOutput](ErrEndpointNotFound)
		}
		return result.Fail[CreateOutput](ErrSaveFailed)
	}

	req, err := uc.saveWithRetry(input)
	if err != nil {
		return result.Fail[CreateOutput](ErrSaveFailed)
	}

	go uc.asyncFlow(req)

	return result.Ok(CreateOutput{Request: req})
}

func buildRequest(input CreateInput) domain.Request {
	return domain.Request{
		Slug:        input.Slug,
		Method:      input.Method,
		Path:        input.Path,
		Headers:     input.Headers,
		QueryParams: input.QueryParams,
		Body:        input.Body,
		SourceIP:    input.SourceIP,
		ReceivedAt:  time.Now().UTC(),
		BodySize:    len(input.Body),
	}
}

func (uc *UseCase) saveWithRetry(input CreateInput) (domain.Request, error) {
	req := buildRequest(input)

	for i := 0; i < maxIDRetries; i++ {
		req.ID = uuid.NewString()

		err := uc.repo.Save(req)
		if err == nil {
			return req, nil
		}
		if !errors.Is(err, ErrIDConflict) {
			return domain.Request{}, err
		}
	}
	return domain.Request{}, ErrIDConflict
}

func (uc *UseCase) asyncFlow(req domain.Request) {
	_ = uc.notifier.NotifyRequest(req)

	attempt, err := uc.forwarder.Forward(req)
	if err != nil {
		attempt = domain.DeliveryAttempt{
			ID:          uuid.NewString(),
			RequestID:   req.ID,
			Delivered:   false,
			Error:       err.Error(),
			AttemptedAt: time.Now().UTC(),
		}
	}

	_ = uc.repo.SaveDeliveryAttempt(attempt)
	_ = uc.notifier.NotifyDelivery(attempt)
}
