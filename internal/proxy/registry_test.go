package proxy_test

import (
	"context"
	"sync"
	"testing"

	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
	"nhooyr.io/websocket"
)

// ── mock ─────────────────────────────────────────────────────────────────────

type mockConn struct{ id int }

func (m *mockConn) Write(_ context.Context, _ websocket.MessageType, _ []byte) error { return nil }
func (m *mockConn) Read(_ context.Context) (websocket.MessageType, []byte, error) {
	return websocket.MessageText, nil, nil
}
func (m *mockConn) Close(_ websocket.StatusCode, _ string) error { return nil }
func (m *mockConn) Ping(_ context.Context) error                 { return nil }

func fakeConn() proxy.Conn { return &mockConn{} }

// ── CLI tests ─────────────────────────────────────────────────────────────────

func TestRegistry_GetCLI_AfterRegister_ReturnsConn(t *testing.T) {
	r := proxy.NewRegistry()
	conn := fakeConn()

	r.RegisterCLI("abc123", conn)
	got, ok := r.GetCLI("abc123")

	if !ok {
		t.Fatal("expected ok to be true")
	}
	if got != conn {
		t.Error("expected returned conn to match registered conn")
	}
}

func TestRegistry_GetCLI_AfterUnregister_ReturnsFalse(t *testing.T) {
	r := proxy.NewRegistry()
	r.RegisterCLI("abc123", fakeConn())
	r.UnregisterCLI("abc123")

	_, ok := r.GetCLI("abc123")

	if ok {
		t.Error("expected ok to be false after unregister")
	}
}

func TestRegistry_GetCLI_DuplicateRegister_OverwritesPrevious(t *testing.T) {
	r := proxy.NewRegistry()
	first := fakeConn()
	second := fakeConn()

	r.RegisterCLI("abc123", first)
	r.RegisterCLI("abc123", second)

	got, ok := r.GetCLI("abc123")

	if !ok {
		t.Fatal("expected ok to be true")
	}
	if got != second {
		t.Error("expected second registration to overwrite first")
	}
}

func TestRegistry_GetCLI_UnknownSlug_ReturnsFalse(t *testing.T) {
	r := proxy.NewRegistry()

	_, ok := r.GetCLI("nonexistent")

	if ok {
		t.Error("expected ok to be false for unknown slug")
	}
}

// ── browser tests ─────────────────────────────────────────────────────────────

func TestRegistry_GetBrowsers_AfterRegister_ReturnsConn(t *testing.T) {
	r := proxy.NewRegistry()
	conn := fakeConn()

	r.RegisterBrowser("abc123", conn)
	conns := r.GetBrowsers("abc123")

	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0] != conn {
		t.Error("expected returned conn to match registered conn")
	}
}

func TestRegistry_GetBrowsers_AfterUnregister_RemovesConn(t *testing.T) {
	r := proxy.NewRegistry()
	conn := fakeConn()

	r.RegisterBrowser("abc123", conn)
	r.UnregisterBrowser("abc123", conn)
	conns := r.GetBrowsers("abc123")

	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

func TestRegistry_GetBrowsers_LastBrowser_CleansUpSlug(t *testing.T) {
	r := proxy.NewRegistry()
	conn := fakeConn()

	r.RegisterBrowser("abc123", conn)
	r.UnregisterBrowser("abc123", conn)

	conns := r.GetBrowsers("abc123")
	if conns == nil {
		t.Error("expected empty slice, got nil")
	}
}

func TestRegistry_GetBrowsers_MultipleConnections_ReturnsAll(t *testing.T) {
	r := proxy.NewRegistry()
	a, b, c := fakeConn(), fakeConn(), fakeConn()

	r.RegisterBrowser("abc123", a)
	r.RegisterBrowser("abc123", b)
	r.RegisterBrowser("abc123", c)

	conns := r.GetBrowsers("abc123")
	if len(conns) != 3 {
		t.Errorf("expected 3 connections, got %d", len(conns))
	}
}

func TestRegistry_GetBrowsers_UnknownSlug_ReturnsEmptySlice(t *testing.T) {
	r := proxy.NewRegistry()

	conns := r.GetBrowsers("nonexistent")

	if conns == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

// ── concurrency ───────────────────────────────────────────────────────────────

func TestRegistry_ConcurrentAccess_NoRace(t *testing.T) {
	r := proxy.NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(3)
		slug := "slug"
		conn := fakeConn()

		go func() {
			defer wg.Done()
			r.RegisterCLI(slug, conn)
		}()
		go func() {
			defer wg.Done()
			r.GetCLI(slug)
		}()
		go func() {
			defer wg.Done()
			r.RegisterBrowser(slug, conn)
			r.GetBrowsers(slug)
			r.UnregisterBrowser(slug, conn)
		}()
	}

	wg.Wait()
}
