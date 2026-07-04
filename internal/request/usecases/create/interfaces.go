package create

import (
	"github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"
	requestdomain "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
)

type Repository interface {
	Save(r requestdomain.Request) error
	SaveDeliveryAttempt(d requestdomain.DeliveryAttempt) error
}


type EndpointRepository interface {
	Get(slug string) (domain.Endpoint, error)
}

type Forwarder interface {
	Forward(r requestdomain.Request) (requestdomain.DeliveryAttempt, error)
}

type Notifier interface {
	NotifyRequest(r requestdomain.Request) error
	NotifyDelivery(d requestdomain.DeliveryAttempt) error
}
