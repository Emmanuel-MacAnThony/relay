package create_test

import (
	"errors"
	"testing"

	"github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/endpoint/usecases/create"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockRepo struct {
	saved        []domain.Endpoint
	saveErr      error
	saveErrAfter int
	callCount    int
}

func (m *mockRepo) Save(e domain.Endpoint) error {
	m.callCount++
	if m.saveErr != nil && m.callCount <= m.saveErrAfter {
		return m.saveErr
	}
	m.saved = append(m.saved, e)
	return nil
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreate_HappyPath_ReturnsEndpoint(t *testing.T) {
	repo := &mockRepo{}
	uc := create.New(repo)

	res := uc.Execute(create.CreateInput{})

	if !res.IsOk() {
		t.Fatalf("expected ok, got: %v", res.Err)
	}
	if res.Value.Endpoint.Slug == "" {
		t.Error("expected non-empty slug")
	}
	if res.Value.Endpoint.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestCreate_HappyPath_SavesEndpoint(t *testing.T) {
	repo := &mockRepo{}
	uc := create.New(repo)

	uc.Execute(create.CreateInput{})

	if len(repo.saved) != 1 {
		t.Fatalf("expected 1 saved endpoint, got %d", len(repo.saved))
	}
	if repo.saved[0].Slug == "" {
		t.Error("expected saved endpoint to have a slug")
	}
}

func TestCreate_SlugConflict_Retries(t *testing.T) {
	repo := &mockRepo{
		saveErr:      create.ErrSlugConflict,
		saveErrAfter: 2,
	}
	uc := create.New(repo)

	res := uc.Execute(create.CreateInput{})

	if !res.IsOk() {
		t.Fatalf("expected ok after retry, got: %v", res.Err)
	}
	if repo.callCount != 3 {
		t.Errorf("expected 3 save attempts, got %d", repo.callCount)
	}
}

func TestCreate_SlugConflict_ExhaustsRetries_ReturnsErrSlugConflict(t *testing.T) {
	repo := &mockRepo{
		saveErr:      create.ErrSlugConflict,
		saveErrAfter: 10,
	}
	uc := create.New(repo)

	res := uc.Execute(create.CreateInput{})

	if res.IsOk() {
		t.Fatal("expected error after exhausted retries")
	}
	if !errors.Is(res.Err, create.ErrSlugConflict) {
		t.Errorf("expected ErrSlugConflict, got %v", res.Err)
	}
}

func TestCreate_SaveFails_ReturnsErrCreateFailed(t *testing.T) {
	repo := &mockRepo{
		saveErr:      errors.New("db error"),
		saveErrAfter: 10,
	}
	uc := create.New(repo)

	res := uc.Execute(create.CreateInput{})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrCreateFailed) {
		t.Errorf("expected ErrCreateFailed, got %v", res.Err)
	}
}

func TestCreate_SlugGenerationFails_ReturnsErrSlugGenerationFailed(t *testing.T) {
	uc := create.NewWithSlugGen(&mockRepo{}, func() (string, error) {
		return "", errors.New("entropy error")
	})

	res := uc.Execute(create.CreateInput{})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrSlugGenerationFailed) {
		t.Errorf("expected ErrSlugGenerationFailed, got %v", res.Err)
	}
}
