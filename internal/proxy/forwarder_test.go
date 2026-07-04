package proxy_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"nhooyr.io/websocket"
)

// ── mock conn ─────────────────────────────────────────────────────────────────
//
// written: Forward sends the request envelope here via Write.
// toRead:  simulateCLI pumps the response envelope here; Read blocks on it.
//
// When readErr is set, Read consumes from written first (so it sequences after
// Write) then immediately returns the error — letting the reader goroutine
// clean up the in-flight pending entry.

type mockForwardConn struct {
	id       int
	writeErr error
	readErr  error
	written  chan []byte
	toRead   chan []byte
}

func newConn(id int) *mockForwardConn {
	return &mockForwardConn{
		id:      id,
		written: make(chan []byte, 1),
		toRead:  make(chan []byte, 1),
	}
}

func (m *mockForwardConn) Write(_ context.Context, _ websocket.MessageType, data []byte) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.written <- data
	return nil
}

func (m *mockForwardConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	if m.readErr != nil {
		select {
		case <-m.written:
		case <-ctx.Done():
		}
		return 0, nil, m.readErr
	}
	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case data := <-m.toRead:
		return websocket.MessageText, data, nil
	}
}

func (m *mockForwardConn) Close(_ websocket.StatusCode, _ string) error { return nil }
func (m *mockForwardConn) Ping(_ context.Context) error                  { return nil }

// simulateCLI waits for a forward_request, extracts the correlation_id, and
// sends back a delivery_result with Delivered: true, StatusCode: 200.
func simulateCLI(conn *mockForwardConn) {
	go func() {
		data := <-conn.written
		var env proxy.ForwardEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			return
		}
		resp := proxy.ResultEnvelope{
			Type:          proxy.TypeDeliveryResult,
			CorrelationID: env.CorrelationID,
			Data: domain.DeliveryAttempt{
				RequestID:   env.Data.ID,
				Delivered:   true,
				StatusCode:  200,
				AttemptedAt: time.Now().UTC(),
			},
		}
		b, _ := json.Marshal(resp)
		conn.toRead <- b
	}()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func storedRequest() domain.Request {
	return domain.Request{
		ID:     "req-123",
		Slug:   "abc123",
		Method: "POST",
		Path:   "/request/abc123",
		Body:   []byte(`{"event":"push"}`),
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestForwarder_NoCLIConnected_ReturnsFailedAttempt(t *testing.T) {
	registry := proxy.NewRegistry()
	f := proxy.NewForwarder(registry, 5*time.Second)

	attempt, err := f.Forward(storedRequest())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if attempt.Delivered {
		t.Error("expected Delivered to be false")
	}
	if attempt.Error != proxy.ErrNoCLI.Error() {
		t.Errorf("expected %q, got %q", proxy.ErrNoCLI.Error(), attempt.Error)
	}
	if attempt.RequestID != "req-123" {
		t.Errorf("expected RequestID req-123, got %s", attempt.RequestID)
	}
}

func TestForwarder_CLIConnected_RespondsSuccessfully(t *testing.T) {
	registry := proxy.NewRegistry()
	f := proxy.NewForwarder(registry, 5*time.Second)

	conn := newConn(1)
	f.RegisterCLI("abc123", conn)
	simulateCLI(conn)

	attempt, err := f.Forward(storedRequest())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !attempt.Delivered {
		t.Error("expected Delivered to be true")
	}
	if attempt.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %d", attempt.StatusCode)
	}
}

func TestForwarder_WriteFails_ReturnsFailedAttempt(t *testing.T) {
	registry := proxy.NewRegistry()
	f := proxy.NewForwarder(registry, 5*time.Second)

	conn := newConn(2)
	conn.writeErr = errors.New("connection lost")
	f.RegisterCLI("abc123", conn)

	attempt, err := f.Forward(storedRequest())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if attempt.Delivered {
		t.Error("expected Delivered to be false")
	}
	if !strings.Contains(attempt.Error, proxy.ErrWriteFailed.Error()) {
		t.Errorf("expected error containing %q, got %q", proxy.ErrWriteFailed.Error(), attempt.Error)
	}
}

func TestForwarder_ReadFails_ReturnsFailedAttempt(t *testing.T) {
	registry := proxy.NewRegistry()
	f := proxy.NewForwarder(registry, 5*time.Second)

	conn := newConn(3)
	conn.readErr = errors.New("connection reset")
	f.RegisterCLI("abc123", conn)

	attempt, err := f.Forward(storedRequest())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if attempt.Delivered {
		t.Error("expected Delivered to be false")
	}
	if !strings.Contains(attempt.Error, proxy.ErrReadFailed.Error()) {
		t.Errorf("expected error containing %q, got %q", proxy.ErrReadFailed.Error(), attempt.Error)
	}
}

func TestForwarder_ReadTimesOut_ReturnsTimeoutMessage(t *testing.T) {
	registry := proxy.NewRegistry()
	f := proxy.NewForwarder(registry, 50*time.Millisecond)

	conn := newConn(4)
	f.RegisterCLI("abc123", conn)
	// no simulateCLI — nothing ever responds

	attempt, err := f.Forward(storedRequest())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if attempt.Delivered {
		t.Error("expected Delivered to be false")
	}
	if attempt.Error != proxy.ErrCLITimeout.Error() {
		t.Errorf("expected %q, got %q", proxy.ErrCLITimeout.Error(), attempt.Error)
	}
}

func TestForwarder_DuplicateInFlight_RejectsSecond(t *testing.T) {
	registry := proxy.NewRegistry()
	f := proxy.NewForwarder(registry, 100*time.Millisecond)

	conn := newConn(5)
	f.RegisterCLI("abc123", conn)

	req := storedRequest()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		f.Forward(req) // blocks until timeout — no simulateCLI
	}()

	// give first Forward time to register in inFlight
	time.Sleep(10 * time.Millisecond)

	second, err := f.Forward(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if second.Delivered {
		t.Error("expected second attempt to be rejected")
	}
	if second.Error != proxy.ErrAlreadyInFlight.Error() {
		t.Errorf("expected %q, got %q", proxy.ErrAlreadyInFlight.Error(), second.Error)
	}

	wg.Wait()
}
