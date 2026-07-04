package create_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	endpointdomain "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/create"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockRepo struct {
	mu            sync.Mutex
	saved         []domain.Request
	savedAttempts []domain.DeliveryAttempt
	saveErr       error
	saveErrAfter  int
	callCount     int
	deliveryOrder   *[]string
	deliveryOrderMu *sync.Mutex
}

func (m *mockRepo) Save(r domain.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.saveErr != nil && m.callCount <= m.saveErrAfter {
		return m.saveErr
	}
	m.saved = append(m.saved, r)
	return nil
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

type mockEndpointRepo struct {
	err error
}

func (m *mockEndpointRepo) Get(slug string) (endpointdomain.Endpoint, error) {
	if m.err != nil {
		return endpointdomain.Endpoint{}, m.err
	}
	return endpointdomain.Endpoint{Slug: slug, CreatedAt: time.Now().UTC()}, nil
}

type mockForwarder struct {
	mu     sync.Mutex
	called bool
	result domain.DeliveryAttempt
	err    error
}

func (m *mockForwarder) Forward(r domain.Request) (domain.DeliveryAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.called = true
	return m.result, m.err
}

type mockNotifier struct {
	mu              sync.Mutex
	requestCalls    []domain.Request
	deliveryCalls   []domain.DeliveryAttempt
	deliveryOrder   *[]string
	deliveryOrderMu *sync.Mutex
}

func (m *mockNotifier) NotifyRequest(r domain.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deliveryOrder != nil {
		m.deliveryOrderMu.Lock()
		*m.deliveryOrder = append(*m.deliveryOrder, "notify-request")
		m.deliveryOrderMu.Unlock()
	}
	m.requestCalls = append(m.requestCalls, r)
	return nil
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

// ── helpers ──────────────────────────────────────────────────────────────────

func validInput() create.CreateInput {
	return create.CreateInput{
		Slug:   "abc123",
		Method: "POST",
		Path:   "/webhook",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body:     []byte(`{"event":"push"}`),
		SourceIP: "192.168.1.1",
	}
}

func okEndpointRepo() *mockEndpointRepo  { return &mockEndpointRepo{} }

func waitForAsync() {
	time.Sleep(50 * time.Millisecond)
}

func newUC(repo *mockRepo, ep *mockEndpointRepo, fwd *mockForwarder, not *mockNotifier) *create.UseCase {
	return create.New(repo, ep, fwd, not)
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestCreate_HappyPath_SavesRequest(t *testing.T) {
	repo := &mockRepo{}
	uc := newUC(repo, okEndpointRepo(), &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(validInput())

	if !res.IsOk() {
		t.Fatalf("expected ok, got err: %v", res.Err)
	}
	if res.Value.Request.ID == "" {
		t.Error("expected non-empty ID")
	}
	if res.Value.Request.Slug != "abc123" {
		t.Errorf("expected slug abc123, got %s", res.Value.Request.Slug)
	}
	if res.Value.Request.ReceivedAt.IsZero() {
		t.Error("expected ReceivedAt to be set")
	}
	if res.Value.Request.BodySize != len(validInput().Body) {
		t.Errorf("expected BodySize %d, got %d", len(validInput().Body), res.Value.Request.BodySize)
	}
	if len(repo.saved) != 1 {
		t.Errorf("expected 1 saved request, got %d", len(repo.saved))
	}
}

func TestCreate_EndpointNotFound_ReturnsErrEndpointNotFound(t *testing.T) {
	ep := &mockEndpointRepo{err: endpointdomain.ErrNotFound}
	uc := newUC(&mockRepo{}, ep, &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(validInput())

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrEndpointNotFound) {
		t.Errorf("expected ErrEndpointNotFound, got %v", res.Err)
	}
}

func TestCreate_EndpointNotFound_RepoSaveNotCalled(t *testing.T) {
	repo := &mockRepo{}
	ep := &mockEndpointRepo{err: endpointdomain.ErrNotFound}
	uc := newUC(repo, ep, &mockForwarder{}, &mockNotifier{})

	uc.Execute(validInput())

	if len(repo.saved) != 0 {
		t.Error("expected Save not to be called when endpoint not found")
	}
}

func TestCreate_EmptySlug_ReturnsErrInvalidInput(t *testing.T) {
	uc := newUC(&mockRepo{}, okEndpointRepo(), &mockForwarder{}, &mockNotifier{})

	input := validInput()
	input.Slug = ""

	res := uc.Execute(input)

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", res.Err)
	}
}

func TestCreate_EmptyMethod_ReturnsErrInvalidInput(t *testing.T) {
	uc := newUC(&mockRepo{}, okEndpointRepo(), &mockForwarder{}, &mockNotifier{})

	input := validInput()
	input.Method = ""

	res := uc.Execute(input)

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", res.Err)
	}
}

func TestCreate_SaveFails_ReturnsErrSaveFailed(t *testing.T) {
	repo := &mockRepo{saveErr: errors.New("db error"), saveErrAfter: 10}
	uc := newUC(repo, okEndpointRepo(), &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(validInput())

	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrSaveFailed) {
		t.Errorf("expected ErrSaveFailed, got %v", res.Err)
	}
}

func TestCreate_SaveFails_ForwarderAndNotifierNotCalled(t *testing.T) {
	repo := &mockRepo{saveErr: errors.New("db error"), saveErrAfter: 10}
	forwarder := &mockForwarder{}
	notifier := &mockNotifier{}
	uc := newUC(repo, okEndpointRepo(), forwarder, notifier)

	uc.Execute(validInput())
	waitForAsync()

	forwarder.mu.Lock()
	called := forwarder.called
	forwarder.mu.Unlock()

	if called {
		t.Error("expected forwarder not to be called on save failure")
	}

	notifier.mu.Lock()
	requestCalls := len(notifier.requestCalls)
	notifier.mu.Unlock()

	if requestCalls > 0 {
		t.Error("expected notifier not to be called on save failure")
	}
}

func TestCreate_IDConflict_Retries(t *testing.T) {
	repo := &mockRepo{saveErr: create.ErrIDConflict, saveErrAfter: 2}
	uc := newUC(repo, okEndpointRepo(), &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(validInput())

	if !res.IsOk() {
		t.Fatalf("expected ok after retry, got: %v", res.Err)
	}
	if repo.callCount != 3 {
		t.Errorf("expected 3 save attempts, got %d", repo.callCount)
	}
}

func TestCreate_IDConflict_ExhaustsRetries_ReturnsErrSaveFailed(t *testing.T) {
	repo := &mockRepo{saveErr: create.ErrIDConflict, saveErrAfter: 10}
	uc := newUC(repo, okEndpointRepo(), &mockForwarder{}, &mockNotifier{})

	res := uc.Execute(validInput())

	if res.IsOk() {
		t.Fatal("expected error after exhausted retries")
	}
	if !errors.Is(res.Err, create.ErrSaveFailed) {
		t.Errorf("expected ErrSaveFailed, got %v", res.Err)
	}
}

func TestCreate_ForwardFailure_DoesNotAffectOutput(t *testing.T) {
	repo := &mockRepo{}
	forwarder := &mockForwarder{err: errors.New("CLI not connected")}
	uc := newUC(repo, okEndpointRepo(), forwarder, &mockNotifier{})

	res := uc.Execute(validInput())

	if !res.IsOk() {
		t.Fatalf("expected ok, got: %v", res.Err)
	}
	if res.Value.Request.ID == "" {
		t.Error("expected request to be returned even when forward fails")
	}
}

func TestCreate_AsyncFlow_ForwarderAndNotifierCalled(t *testing.T) {
	repo := &mockRepo{}
	forwarder := &mockForwarder{}
	notifier := &mockNotifier{}
	uc := newUC(repo, okEndpointRepo(), forwarder, notifier)

	uc.Execute(validInput())
	waitForAsync()

	forwarder.mu.Lock()
	called := forwarder.called
	forwarder.mu.Unlock()

	if !called {
		t.Error("expected forwarder to be called")
	}

	notifier.mu.Lock()
	requestCalls := len(notifier.requestCalls)
	deliveryCalls := len(notifier.deliveryCalls)
	notifier.mu.Unlock()

	if requestCalls != 1 {
		t.Errorf("expected 1 NotifyRequest call, got %d", requestCalls)
	}
	if deliveryCalls != 1 {
		t.Errorf("expected 1 NotifyDelivery call, got %d", deliveryCalls)
	}
}

func TestCreate_PersistBeforeNotify(t *testing.T) {
	var order []string
	var orderMu sync.Mutex
	repo := &mockRepo{deliveryOrder: &order, deliveryOrderMu: &orderMu}
	notifier := &mockNotifier{deliveryOrder: &order, deliveryOrderMu: &orderMu}
	uc := newUC(repo, okEndpointRepo(), &mockForwarder{}, notifier)

	uc.Execute(validInput())
	waitForAsync()

	orderMu.Lock()
	snapshot := make([]string, len(order))
	copy(snapshot, order)
	orderMu.Unlock()

	if len(snapshot) != 3 {
		t.Fatalf("expected 3 events in order, got %d: %v", len(snapshot), snapshot)
	}
	if snapshot[0] != "notify-request" || snapshot[1] != "persist" || snapshot[2] != "notify-delivery" {
		t.Errorf("expected [notify-request persist notify-delivery], got %v", snapshot)
	}
}
