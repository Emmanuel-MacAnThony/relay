package get_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/get"
)


// ── mock ─────────────────────────────────────────────────────────────────────

type mockRepo struct {
	request domain.Request
	err     error
}

func (m *mockRepo) Get(id string) (domain.Request, error) {
	return m.request, m.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func storedRequest() domain.Request {
	return domain.Request{
		ID:         "req-123",
		Slug:       "my-slug",
		Method:     "POST",
		Path:       "/request/my-slug",
		Body:       []byte(`{"event":"push"}`),
		ReceivedAt: time.Now().UTC(),
		BodySize:   16,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGet_HappyPath_ReturnsRequest(t *testing.T) {
	repo := &mockRepo{request: storedRequest()}
	uc := get.New(repo)

	res := uc.Execute(get.GetInput{ID: "req-123"})

	if !res.IsOk() {
		t.Fatalf("expected ok, got: %v", res.Err)
	}
	if res.Value.Request.ID != "req-123" {
		t.Errorf("expected ID req-123, got %s", res.Value.Request.ID)
	}
	if res.Value.Request.Slug != "my-slug" {
		t.Errorf("expected slug my-slug, got %s", res.Value.Request.Slug)
	}
}

func TestGet_EmptyID_ReturnsErrInvalidInput(t *testing.T) {
	uc := get.New(&mockRepo{})

	res := uc.Execute(get.GetInput{ID: ""})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, get.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", res.Err)
	}
}

func TestGet_NotFound_ReturnsErrNotFound(t *testing.T) {
	repo := &mockRepo{err: domain.ErrNotFound}
	uc := get.New(repo)

	res := uc.Execute(get.GetInput{ID: "nonexistent"})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, get.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", res.Err)
	}
}

func TestGet_RepoFails_ReturnsErrGetFailed(t *testing.T) {
	repo := &mockRepo{err: errors.New("db error")}
	uc := get.New(repo)

	res := uc.Execute(get.GetInput{ID: "req-123"})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, get.ErrGetFailed) {
		t.Errorf("expected ErrGetFailed, got %v", res.Err)
	}
}
