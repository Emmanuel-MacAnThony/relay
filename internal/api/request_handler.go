package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/create"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/get"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/list"
	"github.com/Emmanuel-MacAnThony/relay/internal/request/usecases/replay"
	"github.com/Emmanuel-MacAnThony/relay/pkg/logger"
	"github.com/Emmanuel-MacAnThony/relay/pkg/result"
)

type CreateUseCase interface {
	Execute(input create.CreateInput) result.Result[create.CreateOutput]
}

type GetUseCase interface {
	Execute(input get.GetInput) result.Result[get.GetOutput]
}

type ListUseCase interface {
	Execute(input list.ListInput) result.Result[list.ListOutput]
}

type ReplayUseCase interface {
	Execute(input replay.ReplayInput) result.Result[replay.ReplayOutput]
}

type AttemptLister interface {
	ListLatestAttemptsBySlug(slug string) ([]domain.DeliveryAttempt, error)
}

type RequestHandlerDeps struct {
	Create   CreateUseCase
	Get      GetUseCase
	List     ListUseCase
	Replay   ReplayUseCase
	Attempts AttemptLister
}

type RequestHandler struct {
	deps RequestHandlerDeps
	log  *logger.Logger
}

func NewRequestHandler(deps RequestHandlerDeps, log *logger.Logger) *RequestHandler {
	return &RequestHandler{deps: deps, log: log}
}

func (h *RequestHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/hook/{slug}", h.Capture)
	mux.HandleFunc("GET /request/{slug}", h.List)
	mux.HandleFunc("GET /request/{slug}/attempts", h.ListAttempts)
	mux.HandleFunc("GET /request/{slug}/{id}", h.Get)
	mux.HandleFunc("POST /request/{slug}/{id}/replay", h.Replay)
}

// ── response types ────────────────────────────────────────────────────────────

type requestResponse struct {
	ID          string              `json:"id"`
	Slug        string              `json:"slug"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	QueryParams map[string][]string `json:"query_params"`
	Body        string              `json:"body"`
	BodySize    int                 `json:"body_size"`
	ContentType string              `json:"content_type"`
	ReceivedAt  time.Time           `json:"received_at"`
}

func toRequestResponse(req domain.Request) requestResponse {
	ct := ""
	if vals := req.Headers["Content-Type"]; len(vals) > 0 {
		ct = vals[0]
	}
	return requestResponse{
		ID:          req.ID,
		Slug:        req.Slug,
		Method:      req.Method,
		Path:        req.Path,
		Headers:     req.Headers,
		QueryParams: req.QueryParams,
		Body:        base64.StdEncoding.EncodeToString(req.Body),
		BodySize:    req.BodySize,
		ContentType: ct,
		ReceivedAt:  req.ReceivedAt,
	}
}

// ── capture ───────────────────────────────────────────────────────────────────

func (h *RequestHandler) Capture(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read request body", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	res := h.deps.Create.Execute(create.CreateInput{
		Slug:        slug,
		Method:      r.Method,
		Path:        r.URL.Path,
		Headers:     map[string][]string(r.Header),
		QueryParams: map[string][]string(r.URL.Query()),
		Body:        body,
		SourceIP:    r.RemoteAddr,
	})

	if !res.IsOk() {
		switch {
		case errors.Is(res.Err, create.ErrInvalidInput):
			h.log.Warn("invalid webhook input", "slug", slug, "err", res.Err)
			http.Error(w, "bad request", http.StatusBadRequest)
		case errors.Is(res.Err, create.ErrEndpointNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		default:
			h.log.Error("failed to save request", "slug", slug, "err", res.Err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.log.Info("webhook captured", "slug", slug, "id", res.Value.Request.ID)
	w.WriteHeader(http.StatusOK)
}

// ── get ───────────────────────────────────────────────────────────────────────

func (h *RequestHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	res := h.deps.Get.Execute(get.GetInput{ID: id})
	if !res.IsOk() {
		switch {
		case errors.Is(res.Err, get.ErrNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		default:
			h.log.Error("failed to get request", "id", id, "err", res.Err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toRequestResponse(res.Value.Request))
}

// ── replay ────────────────────────────────────────────────────────────────────

func (h *RequestHandler) Replay(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	res := h.deps.Replay.Execute(replay.ReplayInput{ID: id})
	if !res.IsOk() {
		switch {
		case errors.Is(res.Err, replay.ErrNotFound):
			http.Error(w, "not found", http.StatusNotFound)
		default:
			h.log.Error("failed to replay request", "id", id, "err", res.Err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// ── list attempts ─────────────────────────────────────────────────────────────

type deliveryAttemptResponse struct {
	ID              string              `json:"id"`
	RequestID       string              `json:"request_id"`
	Delivered       bool                `json:"delivered"`
	StatusCode      int                 `json:"status_code"`
	Error           string              `json:"error"`
	IsReplay        bool                `json:"is_replay"`
	AttemptedAt     time.Time           `json:"attempted_at"`
	ResponseBody    []byte              `json:"response_body,omitempty"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
}

func (h *RequestHandler) ListAttempts(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	attempts, err := h.deps.Attempts.ListLatestAttemptsBySlug(slug)
	if err != nil {
		h.log.Error("failed to list attempts", "slug", slug, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	responses := make([]deliveryAttemptResponse, len(attempts))
	for i, a := range attempts {
		responses[i] = deliveryAttemptResponse{
			ID:              a.ID,
			RequestID:       a.RequestID,
			Delivered:       a.Delivered,
			StatusCode:      a.StatusCode,
			Error:           a.Error,
			IsReplay:        a.IsReplay,
			AttemptedAt:     a.AttemptedAt,
			ResponseBody:    a.ResponseBody,
			ResponseHeaders: a.ResponseHeaders,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}

// ── list ──────────────────────────────────────────────────────────────────────

func (h *RequestHandler) List(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	res := h.deps.List.Execute(list.ListInput{Slug: slug})
	if !res.IsOk() {
		h.log.Error("failed to list requests", "slug", slug, "err", res.Err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	responses := make([]requestResponse, len(res.Value.Requests))
	for i, req := range res.Value.Requests {
		responses[i] = toRequestResponse(req)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}
