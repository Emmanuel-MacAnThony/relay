package proxy

import (
	"context"
	"sync"

	"nhooyr.io/websocket"
)

// Conn is the interface the registry and forwarder depend on instead of
// *websocket.Conn directly. This keeps both testable without a real WebSocket
// server and decouples us from the concrete library type.
type Conn interface {
	Write(ctx context.Context, typ websocket.MessageType, data []byte) error
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Close(code websocket.StatusCode, reason string) error
	Ping(ctx context.Context) error
}

// Registry is intentionally in-memory. WebSocket connections are transient —
// they cannot survive a server restart regardless of where we store them. If
// the server goes down, every connected CLI and browser must reconnect. Storing
// connection state in Redis or a DB would buy nothing: the objects themselves
// are gone. On restart the CLI reconnects and re-registers, recovering state
// naturally without any persistence layer.
type Registry struct {
	mu sync.RWMutex
	// cli holds exactly one connection per slug. A new RegisterCLI call for
	// the same slug overwrites the previous one — the CLI reconnected.
	cli map[string]Conn
	// browsers uses map[Conn]struct{} (a set) instead of a slice so that
	// unregistering a tab is O(1) rather than O(n). Multiple browser tabs can
	// watch the same slug simultaneously.
	browsers map[string]map[Conn]struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		cli:      make(map[string]Conn),
		browsers: make(map[string]map[Conn]struct{}),
	}
}

func (r *Registry) RegisterCLI(slug string, conn Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cli[slug] = conn
}

func (r *Registry) UnregisterCLI(slug string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cli, slug)
}

func (r *Registry) GetCLI(slug string) (Conn, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, ok := r.cli[slug]
	return conn, ok
}

func (r *Registry) RegisterBrowser(slug string, conn Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.browsers[slug] == nil {
		r.browsers[slug] = make(map[Conn]struct{})
	}
	r.browsers[slug][conn] = struct{}{}
}

func (r *Registry) UnregisterBrowser(slug string, conn Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	conns, ok := r.browsers[slug]
	if !ok {
		return
	}
	delete(conns, conn)
	// clean up the slug entry once the last browser disconnects so the map
	// does not grow unboundedly over the lifetime of the server.
	if len(conns) == 0 {
		delete(r.browsers, slug)
	}
}

func (r *Registry) GetBrowsers(slug string) []Conn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conns := r.browsers[slug]
	result := make([]Conn, 0, len(conns))
	for conn := range conns {
		result = append(result, conn)
	}
	return result
}
