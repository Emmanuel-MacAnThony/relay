package proxy

import "github.com/Emmanuel-MacAnThony/relay/internal/request/domain"

// The relay-CLI protocol uses a simple JSON envelope on every WebSocket message.
// Every message in both directions carries a type field so the reader can
// distinguish message kinds without attempting to deserialise the payload, and
// a correlation_id that ties a CLI response back to the specific Forward call
// that sent the request.
//
// This is necessary because multiple requests can be in-flight over the same
// connection simultaneously — the correlation ID is the only way to route each
// response to the right waiting caller.

const (
	// CLI ↔ server protocol
	TypeForwardRequest = "forward_request"
	TypeDeliveryResult = "delivery_result"

	// server → browser protocol
	TypeRequestCaptured = "request_captured"
	TypeDeliveryAttempt = "delivery_attempt"
)

// ForwardEnvelope is what the relay server sends to the CLI when forwarding an
// incoming webhook request. The CLI reads this, hits the local handler, and
// responds with a ResultEnvelope carrying the same correlation_id.
type ForwardEnvelope struct {
	Type          string         `json:"type"`
	CorrelationID string         `json:"correlation_id"`
	Data          domain.Request `json:"data"`
}

// ResultEnvelope is what the CLI sends back after attempting delivery. The
// relay server matches correlation_id to the waiting Forward call and unblocks
// it with the DeliveryAttempt result.
type ResultEnvelope struct {
	Type          string                 `json:"type"`
	CorrelationID string                 `json:"correlation_id"`
	Data          domain.DeliveryAttempt `json:"data"`
}

// BrowserRequestEnvelope is pushed to dashboard browsers when a new webhook
// arrives. No correlation_id needed — this is a one-way broadcast.
type BrowserRequestEnvelope struct {
	Type string         `json:"type"`
	Data domain.Request `json:"data"`
}

// BrowserDeliveryEnvelope is pushed to dashboard browsers when a delivery
// attempt completes, whether from a live forward or a manual replay.
type BrowserDeliveryEnvelope struct {
	Type string                 `json:"type"`
	Data domain.DeliveryAttempt `json:"data"`
}
