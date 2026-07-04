package replay

import "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"

type Repository interface {
	Get(id string) (domain.Request, error)
	SaveDeliveryAttempt(d domain.DeliveryAttempt) error
}

type Forwarder interface {
	Forward(r domain.Request) (domain.DeliveryAttempt, error)
}

type Notifier interface {
	NotifyDelivery(d domain.DeliveryAttempt) error
}
