package proxy

import (
	"context"
	"encoding/json"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"nhooyr.io/websocket"
)

// Notifier broadcasts events to all browser dashboard connections watching a
// slug. It is write-only — the server never reads from browser connections, so
// there is no read loop and no keepalive needed here.
//
// Delivery is best-effort. If a browser write fails the connection is
// unregistered and the remaining browsers still receive the event. A browser
// that missed a push loads current state from the DB on reconnect.
type Notifier struct {
	registry *Registry
}

func NewNotifier(registry *Registry) *Notifier {
	return &Notifier{registry: registry}
}

func (n *Notifier) NotifyRequest(r domain.Request) error {
	data, err := json.Marshal(BrowserRequestEnvelope{
		Type: TypeRequestCaptured,
		Data: r,
	})
	if err != nil {
		return err
	}
	n.broadcast(r.Slug, data)
	return nil
}

func (n *Notifier) NotifyDelivery(d domain.DeliveryAttempt) error {
	data, err := json.Marshal(BrowserDeliveryEnvelope{
		Type: TypeDeliveryAttempt,
		Data: d,
	})
	if err != nil {
		return err
	}
	n.broadcast(d.Slug, data)
	return nil
}

// broadcast writes data to every browser connected to slug. A failed write
// means the browser disconnected — unregister it and continue to the rest.
func (n *Notifier) broadcast(slug string, data []byte) {
	for _, conn := range n.registry.GetBrowsers(slug) {
		if err := conn.Write(context.Background(), websocket.MessageText, data); err != nil {
			n.registry.UnregisterBrowser(slug, conn)
		}
	}
}
