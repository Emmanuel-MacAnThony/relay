package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/proxy"
	"nhooyr.io/websocket"
)

const (
	browserKeepaliveInterval = 30 * time.Second
	browserKeepaliveTimeout  = 5 * time.Second
)

type WSHandler struct {
	registry  *proxy.Registry
	forwarder *proxy.Forwarder
}

func NewWSHandler(registry *proxy.Registry, forwarder *proxy.Forwarder) *WSHandler {
	return &WSHandler{registry: registry, forwarder: forwarder}
}

func (h *WSHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ws/cli/{slug}", h.CLI)
	mux.HandleFunc("GET /ws/browser/{slug}", h.Browser)
}

func (h *WSHandler) CLI(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	h.forwarder.RegisterCLI(slug, conn)
}

func (h *WSHandler) Browser(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	h.registry.RegisterBrowser(slug, conn)
	defer h.registry.UnregisterBrowser(slug, conn)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.browserKeepalive(ctx, conn, slug)

	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}

func (h *WSHandler) browserKeepalive(ctx context.Context, conn proxy.Conn, slug string) {
	ticker := time.NewTicker(browserKeepaliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, browserKeepaliveTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				if ctx.Err() == nil {
					conn.Close(websocket.StatusGoingAway, "keepalive failed")
				}
				return
			}
		}
	}
}
