package list_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/list"
)

// ── mock ─────────────────────────────────────────────────────────────────────

type mockRepo struct {
	requests []domain.Request
	err      error
}

func (m *mockRepo) List(slug string) ([]domain.Request, error) {
	return m.requests, m.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func makeRequests(slug string, n int) []domain.Request {
	reqs := make([]domain.Request, n)
	for i := range reqs {
		reqs[i] = domain.Request{
			ID:         "req-" + string(rune('0'+i)),
			Slug:       slug,
			Method:     "POST",
			Path:       "/request/" + slug,
			ReceivedAt: time.Now().UTC(),
		}
	}
	return reqs
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestList_HappyPath_ReturnsRequests(t *testing.T) {
	repo := &mockRepo{requests: makeRequests("my-slug", 3)}
	uc := list.New(repo)

	res := uc.Execute(list.ListInput{Slug: "my-slug"})

	if !res.IsOk() {
		t.Fatalf("expected ok, got: %v", res.Err)
	}
	if len(res.Value.Requests) != 3 {
		t.Errorf("expected 3 requests, got %d", len(res.Value.Requests))
	}
}

func TestList_EmptySlug_ReturnsErrInvalidInput(t *testing.T) {
	uc := list.New(&mockRepo{})

	res := uc.Execute(list.ListInput{Slug: ""})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, list.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", res.Err)
	}
}

func TestList_NoRequests_ReturnsEmptyList(t *testing.T) {
	repo := &mockRepo{requests: []domain.Request{}}
	uc := list.New(repo)

	res := uc.Execute(list.ListInput{Slug: "my-slug"})

	if !res.IsOk() {
		t.Fatal("expected ok, got error")
	}
	if len(res.Value.Requests) != 0 {
		t.Errorf("expected empty list, got %d", len(res.Value.Requests))
	}
}

func TestList_RepoFails_ReturnsErrListFailed(t *testing.T) {
	repo := &mockRepo{err: errors.New("db error")}
	uc := list.New(repo)

	res := uc.Execute(list.ListInput{Slug: "my-slug"})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, list.ErrListFailed) {
		t.Errorf("expected ErrListFailed, got %v", res.Err)
	}
}
