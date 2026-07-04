package proxy_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"nhooyr.io/websocket"
)

// ── mock conn ─────────────────────────────────────────────────────────────────

type mockBrowserConn struct {
	id       int
	writeErr error
	received [][]byte
}

func (m *mockBrowserConn) Write(_ context.Context, _ websocket.MessageType, data []byte) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.received = append(m.received, data)
	return nil
}

func (m *mockBrowserConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	<-ctx.Done()
	return 0, nil, ctx.Err()
}

func (m *mockBrowserConn) Close(_ websocket.StatusCode, _ string) error { return nil }
func (m *mockBrowserConn) Ping(_ context.Context) error                  { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

type browserEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func parseEvent(t *testing.T, data []byte) browserEnvelope {
	t.Helper()
	var env browserEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("failed to parse browser envelope: %v", err)
	}
	return env
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestNotifier_NoBrowsers_Noop(t *testing.T) {
	registry := proxy.NewRegistry()
	n := proxy.NewNotifier(registry)

	err := n.NotifyRequest(domain.Request{ID: "req-1", Slug: "abc123"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestNotifier_NotifyRequest_SendsRequestCapturedEnvelope(t *testing.T) {
	registry := proxy.NewRegistry()
	n := proxy.NewNotifier(registry)
	conn := &mockBrowserConn{id: 1}
	registry.RegisterBrowser("abc123", conn)

	err := n.NotifyRequest(domain.Request{ID: "req-1", Slug: "abc123", Method: "POST"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(conn.received) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conn.received))
	}
	if env := parseEvent(t, conn.received[0]); env.Type != proxy.TypeRequestCaptured {
		t.Errorf("expected type %q, got %q", proxy.TypeRequestCaptured, env.Type)
	}
}

func TestNotifier_NotifyDelivery_SendsDeliveryAttemptEnvelope(t *testing.T) {
	registry := proxy.NewRegistry()
	n := proxy.NewNotifier(registry)
	conn := &mockBrowserConn{id: 2}
	registry.RegisterBrowser("abc123", conn)

	err := n.NotifyDelivery(domain.DeliveryAttempt{
		ID:        "att-1",
		RequestID: "req-1",
		Slug:      "abc123",
		Delivered: true,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(conn.received) != 1 {
		t.Fatalf("expected 1 message, got %d", len(conn.received))
	}
	if env := parseEvent(t, conn.received[0]); env.Type != proxy.TypeDeliveryAttempt {
		t.Errorf("expected type %q, got %q", proxy.TypeDeliveryAttempt, env.Type)
	}
}

func TestNotifier_MultipleBrowsers_AllReceive(t *testing.T) {
	registry := proxy.NewRegistry()
	n := proxy.NewNotifier(registry)
	a := &mockBrowserConn{id: 3}
	b := &mockBrowserConn{id: 4}
	c := &mockBrowserConn{id: 5}
	registry.RegisterBrowser("abc123", a)
	registry.RegisterBrowser("abc123", b)
	registry.RegisterBrowser("abc123", c)

	err := n.NotifyRequest(domain.Request{ID: "req-1", Slug: "abc123"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for i, conn := range []*mockBrowserConn{a, b, c} {
		if len(conn.received) != 1 {
			t.Errorf("browser %d: expected 1 message, got %d", i, len(conn.received))
		}
	}
}

func TestNotifier_WriteFails_UnregistersBrowserContinuesOthers(t *testing.T) {
	registry := proxy.NewRegistry()
	n := proxy.NewNotifier(registry)
	bad := &mockBrowserConn{id: 6, writeErr: errors.New("connection closed")}
	good := &mockBrowserConn{id: 7}
	registry.RegisterBrowser("abc123", bad)
	registry.RegisterBrowser("abc123", good)

	err := n.NotifyRequest(domain.Request{ID: "req-1", Slug: "abc123"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(good.received) != 1 {
		t.Errorf("expected good browser to receive 1 message, got %d", len(good.received))
	}
	for _, conn := range registry.GetBrowsers("abc123") {
		if conn == bad {
			t.Error("expected bad browser to be unregistered after write failure")
		}
	}
}
