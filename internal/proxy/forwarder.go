package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

const (
	keepaliveInterval = 30 * time.Second
	keepaliveTimeout  = 5 * time.Second
)

// pendingEntry tracks a single in-flight Forward call. slug and requestID are
// stored alongside the channel so the readLoop can sweep and fail all pending
// entries for a slug when the CLI connection drops — without needing a separate
// index from slug to correlation IDs.
type pendingEntry struct {
	ch        chan domain.DeliveryAttempt
	slug      string
	requestID string
}

// Forwarder sends requests to a connected CLI over WebSocket and waits for the
// delivery result. It owns the read side of every CLI connection — one
// goroutine per connection loops on Read and dispatches responses to the
// correct caller via a correlation ID.
//
// Concurrent Forward calls for the same slug are safe: each gets its own
// correlation ID and channel. Reads are serialised through the loop; the
// pending map fans the responses back out to the right waiters.
type Forwarder struct {
	registry *Registry
	timeout  time.Duration
	mu       sync.Mutex
	// pending maps a correlation ID to the Forward call waiting for its result.
	// Both Forward (writes new entries, deletes on timeout/write-error) and the
	// readLoop (deletes on response or CLI disconnect) access this map, hence
	// the mutex. Even accesses on different keys require a lock because Go maps
	// are not safe for concurrent use — the underlying hash table can resize.
	pending map[string]*pendingEntry
	// inFlight prevents the same request from being forwarded twice at the same
	// time. Without this, a user double-triggering replay would send the request
	// to the CLI twice and race on Read for the same response slot.
	inFlight map[string]struct{}
}

func NewForwarder(registry *Registry, timeout time.Duration) *Forwarder {
	return &Forwarder{
		registry: registry,
		timeout:  timeout,
		pending:  make(map[string]*pendingEntry),
		inFlight: make(map[string]struct{}),
	}
}

// RegisterCLI wires a CLI connection into the registry and immediately starts
// the read loop for it. Callers should use this instead of registry.RegisterCLI
// directly so the loop is always running when a connection is live.
func (f *Forwarder) RegisterCLI(slug string, conn Conn) {
	f.registry.RegisterCLI(slug, conn)
	go f.readLoop(slug, conn)
}

func (f *Forwarder) Forward(req domain.Request) (domain.DeliveryAttempt, error) {
	f.mu.Lock()
	if _, exists := f.inFlight[req.ID]; exists {
		f.mu.Unlock()
		return f.failedAttempt(req.ID, ErrAlreadyInFlight), nil
	}
	f.inFlight[req.ID] = struct{}{}
	f.mu.Unlock()

	// Always remove from inFlight when Forward exits, regardless of outcome,
	// so a subsequent replay of the same request is allowed once this one settles.
	defer func() {
		f.mu.Lock()
		delete(f.inFlight, req.ID)
		f.mu.Unlock()
	}()

	conn, ok := f.registry.GetCLI(req.Slug)
	if !ok {
		return f.failedAttempt(req.ID, ErrNoCLI), nil
	}

	correlationID := uuid.New().String()
	// Buffered size 1 so the readLoop can send without blocking even if Forward
	// has already returned due to a timeout — the value is simply dropped.
	ch := make(chan domain.DeliveryAttempt, 1)

	f.mu.Lock()
	f.pending[correlationID] = &pendingEntry{
		ch:        ch,
		slug:      req.Slug,
		requestID: req.ID,
	}
	f.mu.Unlock()

	data, err := json.Marshal(ForwardEnvelope{
		Type:          TypeForwardRequest,
		CorrelationID: correlationID,
		Data:          req,
	})
	if err != nil {
		f.mu.Lock()
		delete(f.pending, correlationID)
		f.mu.Unlock()
		return f.failedAttempt(req.ID, fmt.Errorf("%w: %s", ErrMarshalFailed, err)), nil
	}

	if err := conn.Write(context.Background(), websocket.MessageText, data); err != nil {
		f.mu.Lock()
		delete(f.pending, correlationID)
		f.mu.Unlock()
		return f.failedAttempt(req.ID, fmt.Errorf("%w: %s", ErrWriteFailed, err)), nil
	}

	select {
	case attempt := <-ch:
		// readLoop already deleted the pending entry before sending here.
		return attempt, nil
	case <-time.After(f.timeout):
		// No response within the deadline. The readLoop may still be running
		// and could deliver a response after we return — the buffered channel
		// absorbs the send so the readLoop never blocks, and the value is
		// discarded since nobody is reading from ch anymore.
		f.mu.Lock()
		delete(f.pending, correlationID)
		f.mu.Unlock()
		return f.failedAttempt(req.ID, ErrCLITimeout), nil
	}
}

// readLoop is the sole reader for a CLI connection. Running one goroutine per
// connection means reads are serialised per connection while multiple Forward
// calls can still be in-flight simultaneously — the pending map routes each
// response to the right waiting caller by correlation ID.
//
// A cancellable context is created here and passed to the keepalive goroutine.
// When readLoop exits for any reason, defer cancel() fires and the keepalive
// goroutine exits via ctx.Done() — both goroutines are tied to the same lifetime.
func (f *Forwarder) readLoop(slug string, conn Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go f.keepalive(ctx, conn)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			// CLI disconnected. Unregister so new requests are not routed here,
			// then fail every pending Forward waiting on this connection. The
			// CLI must reconnect and re-register to receive requests again.
			f.registry.UnregisterCLI(slug)
			f.mu.Lock()
			for corrID, entry := range f.pending {
				if entry.slug == slug {
					entry.ch <- f.failedAttempt(entry.requestID, fmt.Errorf("%w: %s", ErrReadFailed, err))
					delete(f.pending, corrID)
				}
			}
			f.mu.Unlock()
			return
		}

		var env ResultEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			// Malformed message — skip rather than crash the loop. The CLI
			// may have sent a heartbeat or non-JSON frame.
			continue
		}
		if env.Type != TypeDeliveryResult {
			// Unexpected message type — the protocol may grow new types in
			// future; ignore unknown ones rather than treating them as errors.
			continue
		}

		// Delete inside the lock before sending so the map is clean before
		// the Forward goroutine can observe the result. Re-acquiring the lock
		// after the send would leave a window where the entry exists but is
		// no longer valid.
		f.mu.Lock()
		entry, ok := f.pending[env.CorrelationID]
		if ok {
			delete(f.pending, env.CorrelationID)
		}
		f.mu.Unlock()

		if ok {
			entry.ch <- env.Data
		}
	}
}

// keepalive pings the CLI on a fixed interval to detect silently dead
// connections. Without this, a connection lost mid-flight (network partition,
// SIGKILL) leaves the readLoop goroutine parked on Read indefinitely — the OS
// TCP timeout can be hours. A failed ping closes the connection, which causes
// Read to error and the readLoop to exit through its normal cleanup path.
//
// keepalive exits when ctx is cancelled (readLoop is done) or when a ping
// fails. It never calls Close if ctx is already cancelled — readLoop already
// handled cleanup in that case.
func (f *Forwarder) keepalive(ctx context.Context, conn Conn) {
	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, keepaliveTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				if ctx.Err() == nil {
					// readLoop is still running — connection is dead, close it
					// so Read unblocks and readLoop exits through its normal path.
					conn.Close(websocket.StatusGoingAway, "keepalive failed")
				}
				return
			}
		}
	}
}

func (f *Forwarder) failedAttempt(requestID string, err error) domain.DeliveryAttempt {
	return domain.DeliveryAttempt{
		ID:          uuid.New().String(),
		RequestID:   requestID,
		Delivered:   false,
		Error:       err.Error(),
		AttemptedAt: time.Now().UTC(),
	}
}
