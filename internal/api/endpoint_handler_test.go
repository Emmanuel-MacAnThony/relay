package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Emmanuel-MacAnThony/relay/internal/api"
	"github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"
	endpointcreate "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/usecases/create"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
)

// ── mock ─────────────────────────────────────────────────────────────────────

type mockEndpointCreate struct {
	result result.Result[endpointcreate.CreateOutput]
}

func (m *mockEndpointCreate) Execute(_ endpointcreate.CreateInput) result.Result[endpointcreate.CreateOutput] {
	return m.result
}

// ── setup ─────────────────────────────────────────────────────────────────────

func newEndpointTestServer(uc api.EndpointCreateUseCase) *httptest.Server {
	h := api.NewEndpointHandler(api.EndpointHandlerDeps{Create: uc}, "https://relay.dev", logger.New())
	router := api.NewRouter(api.RouterDeps{
		Request:  api.NewRequestHandler(api.RequestHandlerDeps{}, logger.New()),
		Endpoint: h,
	})
	return httptest.NewServer(router)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestEndpointCreate_Returns201_WithSlugAndURL(t *testing.T) {
	uc := &mockEndpointCreate{result: result.Ok(endpointcreate.CreateOutput{
		Endpoint: domain.Endpoint{Slug: "abc123"},
	})}

	srv := newEndpointTestServer(uc)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/endpoint", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var body struct {
		Slug string `json:"slug"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Slug != "abc123" {
		t.Errorf("expected slug abc123, got %s", body.Slug)
	}
	if body.URL != "https://relay.dev/hook/abc123" {
		t.Errorf("expected url https://relay.dev/hook/abc123, got %s", body.URL)
	}
}

func TestEndpointCreate_Returns409_OnSlugConflict(t *testing.T) {
	uc := &mockEndpointCreate{
		result: result.Fail[endpointcreate.CreateOutput](endpointcreate.ErrSlugConflict),
	}

	srv := newEndpointTestServer(uc)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/endpoint", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestEndpointCreate_Returns500_OnCreateFailed(t *testing.T) {
	uc := &mockEndpointCreate{
		result: result.Fail[endpointcreate.CreateOutput](endpointcreate.ErrCreateFailed),
	}

	srv := newEndpointTestServer(uc)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/endpoint", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
