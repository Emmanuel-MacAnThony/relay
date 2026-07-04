package replay_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/replay"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockRepo struct {
	mu            sync.Mutex
	request       domain.Request
	getErr        error
	savedAttempts []domain.DeliveryAttempt
	deliveryOrder   *[]string
	deliveryOrderMu *sync.Mutex
}

func (m *mockRepo) Get(id string) (domain.Request, error) {
	return m.request, m.getErr
}

func (m *mockRepo) SaveDeliveryAttempt(d domain.DeliveryAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deliveryOrder != nil {
		m.deliveryOrderMu.Lock()
		*m.deliveryOrder = append(*m.deliveryOrder, "persist")
		m.deliveryOrderMu.Unlock()
	}
	m.savedAttempts = append(m.savedAttempts, d)
	return nil
}

type mockForwarder struct {
	mu     sync.Mutex
	called bool
	req    domain.Request
	result domain.DeliveryAttempt
	err    error
}

func (m *mockForwarder) Forward(r domain.Request) (domain.DeliveryAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.called = true
	m.req = r
	return m.result, m.err
}

type mockNotifier struct {
	mu              sync.Mutex
	deliveryCalls   []domain.DeliveryAttempt
	deliveryOrder   *[]string
	deliveryOrderMu *sync.Mutex
}

func (m *mockNotifier) NotifyDelivery(d domain.DeliveryAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deliveryOrder != nil {
		m.deliveryOrderMu.Lock()
		*m.deliveryOrder = append(*m.deliveryOrder, "notify-delivery")
		m.deliveryOrderMu.Unlock()
	}
	m.deliveryCalls = append(m.deliveryCalls, d)
	return nil
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

func waitForAsync() {
	time.Sleep(50 * time.Millisecond)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestReplay_HappyPath_ReturnsOk(t *testing.T) {
	repo := &mockRepo{request: storedRequest()}
	uc := replay.New(repo, &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(replay.ReplayInput{ID: "req-123"})

	if !res.IsOk() {
		t.Fatalf("expected ok, got: %v", res.Err)
	}
}

func TestReplay_EmptyID_ReturnsErrInvalidInput(t *testing.T) {
	uc := replay.New(&mockRepo{}, &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(replay.ReplayInput{ID: ""})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, replay.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", res.Err)
	}
}

func TestReplay_RequestNotFound_ReturnsErrNotFound(t *testing.T) {
	repo := &mockRepo{getErr: domain.ErrNotFound}
	uc := replay.New(repo, &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(replay.ReplayInput{ID: "nonexistent"})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, replay.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", res.Err)
	}
}

func TestReplay_GetFails_ReturnsErrReplayFailed(t *testing.T) {
	repo := &mockRepo{getErr: errors.New("db error")}
	uc := replay.New(repo, &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(replay.ReplayInput{ID: "req-123"})

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, replay.ErrReplayFailed) {
		t.Errorf("expected ErrReplayFailed, got %v", res.Err)
	}
}

func TestReplay_AsyncFlow_ForwarderAndSaveAndNotifyCalled(t *testing.T) {
	repo := &mockRepo{request: storedRequest()}
	forwarder := &mockForwarder{}
	notifier := &mockNotifier{}
	uc := replay.New(repo, forwarder, notifier)

	uc.Execute(replay.ReplayInput{ID: "req-123"})
	waitForAsync()

	forwarder.mu.Lock()
	called := forwarder.called
	forwarder.mu.Unlock()

	if !called {
		t.Error("expected forwarder to be called")
	}

	repo.mu.Lock()
	attempts := len(repo.savedAttempts)
	repo.mu.Unlock()

	if attempts != 1 {
		t.Errorf("expected 1 saved delivery attempt, got %d", attempts)
	}

	notifier.mu.Lock()
	deliveryCalls := len(notifier.deliveryCalls)
	notifier.mu.Unlock()

	if deliveryCalls != 1 {
		t.Errorf("expected 1 NotifyDelivery call, got %d", deliveryCalls)
	}
}

func TestReplay_AttemptHasIsReplayTrue(t *testing.T) {
	repo := &mockRepo{request: storedRequest()}
	uc := replay.New(repo, &mockForwarder{}, &mockNotifier{})

	uc.Execute(replay.ReplayInput{ID: "req-123"})
	waitForAsync()

	repo.mu.Lock()
	attempts := repo.savedAttempts
	repo.mu.Unlock()

	if len(attempts) == 0 {
		t.Fatal("expected a saved delivery attempt")
	}
	if !attempts[0].IsReplay {
		t.Error("expected IsReplay to be true")
	}
}

func TestReplay_ForwardFails_AttemptSavedWithDeliveredFalse(t *testing.T) {
	repo := &mockRepo{request: storedRequest()}
	forwarder := &mockForwarder{err: errors.New("CLI not connected")}
	uc := replay.New(repo, forwarder, &mockNotifier{})

	uc.Execute(replay.ReplayInput{ID: "req-123"})
	waitForAsync()

	repo.mu.Lock()
	attempts := repo.savedAttempts
	repo.mu.Unlock()

	if len(attempts) == 0 {
		t.Fatal("expected a saved delivery attempt even on forward failure")
	}
	if attempts[0].Delivered {
		t.Error("expected Delivered to be false on forward failure")
	}
	if attempts[0].Error == "" {
		t.Error("expected Error to be set on forward failure")
	}
}

func TestReplay_PersistBeforeNotify(t *testing.T) {
	var order []string
	var orderMu sync.Mutex
	repo := &mockRepo{request: storedRequest(), deliveryOrder: &order, deliveryOrderMu: &orderMu}
	notifier := &mockNotifier{deliveryOrder: &order, deliveryOrderMu: &orderMu}
	uc := replay.New(repo, &mockForwarder{}, notifier)

	uc.Execute(replay.ReplayInput{ID: "req-123"})
	waitForAsync()

	orderMu.Lock()
	snapshot := make([]string, len(order))
	copy(snapshot, order)
	orderMu.Unlock()

	if len(snapshot) != 2 {
		t.Fatalf("expected 2 events in order, got %d: %v", len(snapshot), snapshot)
	}
	if snapshot[0] != "persist" || snapshot[1] != "notify-delivery" {
		t.Errorf("expected [persist notify-delivery], got %v", snapshot)
	}
}
