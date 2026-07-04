package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/api"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/create"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/get"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/list"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/replay"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockCreate struct {
	result result.Result[create.CreateOutput]
}

func (m *mockCreate) Execute(_ create.CreateInput) result.Result[create.CreateOutput] {
	return m.result
}

type mockGet struct {
	result result.Result[get.GetOutput]
}

func (m *mockGet) Execute(_ get.GetInput) result.Result[get.GetOutput] {
	return m.result
}

type mockList struct {
	result result.Result[list.ListOutput]
}

func (m *mockList) Execute(_ list.ListInput) result.Result[list.ListOutput] {
	return m.result
}

// ── setup ─────────────────────────────────────────────────────────────────────

func newTestServer(deps api.RequestHandlerDeps) *httptest.Server {
	h := api.NewRequestHandler(deps, logger.New())
	e := api.NewEndpointHandler(api.EndpointHandlerDeps{}, "http://localhost", logger.New())
	return httptest.NewServer(api.NewRouter(api.RouterDeps{Request: h, Endpoint: e}))
}

func storedRequest() domain.Request {
	return domain.Request{
		ID:         "req-123",
		Slug:       "my-slug",
		Method:     "POST",
		Path:       "/hook/my-slug",
		Body:       []byte(`{"event":"push"}`),
		ReceivedAt: time.Now().UTC(),
		BodySize:   16,
	}
}

// ── capture tests ─────────────────────────────────────────────────────────────

func TestCapture_Returns200_OnSuccess(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Create: &mockCreate{result: result.Ok(create.CreateOutput{
			Request: domain.Request{ID: "abc123", Slug: "my-slug"},
		})},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/hook/my-slug", "application/json", strings.NewReader(`{"event":"push"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCapture_Returns400_OnInvalidInput(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Create: &mockCreate{result: result.Fail[create.CreateOutput](create.ErrInvalidInput)},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/hook/my-slug", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCapture_Returns500_OnSaveFailure(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Create: &mockCreate{result: result.Fail[create.CreateOutput](create.ErrSaveFailed)},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/hook/my-slug", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// ── get tests ─────────────────────────────────────────────────────────────────

func TestGet_Returns200_OnSuccess(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Get: &mockGet{result: result.Ok(get.GetOutput{Request: storedRequest()})},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/request/my-slug/req-123")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGet_Returns404_OnNotFound(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Get: &mockGet{result: result.Fail[get.GetOutput](get.ErrNotFound)},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/request/my-slug/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGet_Returns500_OnGetFailure(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Get: &mockGet{result: result.Fail[get.GetOutput](get.ErrGetFailed)},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/request/my-slug/req-123")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// ── list tests ────────────────────────────────────────────────────────────────

func TestList_Returns200_WithRequests(t *testing.T) {
	deps := api.RequestHandlerDeps{
		List: &mockList{result: result.Ok(list.ListOutput{
			Requests: []domain.Request{storedRequest()},
		})},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/request/my-slug")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestList_Returns200_WithEmptyList(t *testing.T) {
	deps := api.RequestHandlerDeps{
		List: &mockList{result: result.Ok(list.ListOutput{
			Requests: []domain.Request{},
		})},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/request/my-slug")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestList_Returns500_OnListFailure(t *testing.T) {
	deps := api.RequestHandlerDeps{
		List: &mockList{result: result.Fail[list.ListOutput](list.ErrListFailed)},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/request/my-slug")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// ── replay tests ──────────────────────────────────────────────────────────────

type mockReplay struct {
	result result.Result[replay.ReplayOutput]
}

func (m *mockReplay) Execute(_ replay.ReplayInput) result.Result[replay.ReplayOutput] {
	return m.result
}

func TestReplay_Returns202_OnSuccess(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Replay: &mockReplay{result: result.Ok(replay.ReplayOutput{})},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/request/my-slug/req-123/replay", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202, got %d", resp.StatusCode)
	}
}

func TestReplay_Returns404_OnNotFound(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Replay: &mockReplay{result: result.Fail[replay.ReplayOutput](replay.ErrNotFound)},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/request/my-slug/nonexistent/replay", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestReplay_Returns500_OnReplayFailure(t *testing.T) {
	deps := api.RequestHandlerDeps{
		Replay: &mockReplay{result: result.Fail[replay.ReplayOutput](replay.ErrReplayFailed)},
	}

	srv := newTestServer(deps)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/request/my-slug/req-123/replay", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
