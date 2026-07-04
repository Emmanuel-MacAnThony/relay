package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	endpointcreate "github.com/Emmanuel-MacAnThony/relay/internal/endpoint/usecases/create"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
)

type EndpointCreateUseCase interface {
	Execute(input endpointcreate.CreateInput) result.Result[endpointcreate.CreateOutput]
}

type EndpointHandlerDeps struct {
	Create EndpointCreateUseCase
}

type EndpointHandler struct {
	deps    EndpointHandlerDeps
	baseURL string
	log     *logger.Logger
}

func NewEndpointHandler(deps EndpointHandlerDeps, baseURL string, log *logger.Logger) *EndpointHandler {
	return &EndpointHandler{deps: deps, baseURL: baseURL, log: log}
}

func (h *EndpointHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /endpoint", h.Create)
}

// ── response types ────────────────────────────────────────────────────────────

type endpointResponse struct {
	Slug      string    `json:"slug"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// ── create ────────────────────────────────────────────────────────────────────

func (h *EndpointHandler) Create(w http.ResponseWriter, r *http.Request) {
	res := h.deps.Create.Execute(endpointcreate.CreateInput{})
	if !res.IsOk() {
		switch {
		case errors.Is(res.Err, endpointcreate.ErrSlugConflict):
			http.Error(w, "conflict", http.StatusConflict)
		default:
			h.log.Error("failed to create endpoint", "err", res.Err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	e := res.Value.Endpoint
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(endpointResponse{
		Slug:      e.Slug,
		URL:       e.URL(h.baseURL),
		CreatedAt: e.CreatedAt,
	})
}
