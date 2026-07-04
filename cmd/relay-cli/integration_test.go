package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/api"
	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func setupRelay(t *testing.T) (*proxy.Registry, *proxy.Forwarder, string) {
	t.Helper()
	registry := proxy.NewRegistry()
	forwarder := proxy.NewForwarder(registry, 5*time.Second)
	mux := http.NewServeMux()
	api.NewWSHandler(registry, forwarder).RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return registry, forwarder, "ws" + strings.TrimPrefix(srv.URL, "http")
}

func startCLI(t *testing.T, serverURL, slug, target string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go NewClient(serverURL, slug, NewExecutor(target), logger.New()).Run(ctx)
}

func waitForCLI(t *testing.T, registry *proxy.Registry, slug string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := registry.GetCLI(slug); ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("CLI did not connect within 2 seconds")
}

type captured struct {
	method string
	body   []byte
}

func localServer(t *testing.T, status int) (*httptest.Server, <-chan captured) {
	t.Helper()
	ch := make(chan captured, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ch <- captured{method: r.Method, body: body}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, ch
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCLI_HappyPath(t *testing.T) {
	local, requests := localServer(t, http.StatusOK)
	registry, forwarder, wsURL := setupRelay(t)
	startCLI(t, wsURL, "test-slug", local.URL)
	waitForCLI(t, registry, "test-slug")

	attempt, err := forwarder.Forward(domain.Request{
		ID:     "req-1",
		Slug:   "test-slug",
		Method: "POST",
		Body:   []byte(`{"event":"push"}`),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !attempt.Delivered {
		t.Errorf("expected delivered=true, error: %s", attempt.Error)
	}
	if attempt.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", attempt.StatusCode)
	}

	select {
	case req := <-requests:
		if req.method != "POST" {
			t.Errorf("expected method POST, got %s", req.method)
		}
		if string(req.body) != `{"event":"push"}` {
			t.Errorf("unexpected body: %s", req.body)
		}
	case <-time.After(time.Second):
		t.Error("local server never received request")
	}
}

func TestCLI_NonOKResponse(t *testing.T) {
	local, _ := localServer(t, http.StatusUnprocessableEntity)
	registry, forwarder, wsURL := setupRelay(t)
	startCLI(t, wsURL, "test-slug", local.URL)
	waitForCLI(t, registry, "test-slug")

	attempt, err := forwarder.Forward(domain.Request{
		ID:     "req-2",
		Slug:   "test-slug",
		Method: "POST",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempt.Delivered {
		t.Error("expected delivered=false for non-2xx response")
	}
	if attempt.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", attempt.StatusCode)
	}
}

func TestCLI_LocalServerUnreachable(t *testing.T) {
	registry, forwarder, wsURL := setupRelay(t)
	startCLI(t, wsURL, "test-slug", "http://localhost:1")
	waitForCLI(t, registry, "test-slug")

	attempt, err := forwarder.Forward(domain.Request{
		ID:     "req-3",
		Slug:   "test-slug",
		Method: "POST",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempt.Delivered {
		t.Error("expected delivered=false for unreachable target")
	}
	if attempt.Error == "" {
		t.Error("expected non-empty error")
	}
}
