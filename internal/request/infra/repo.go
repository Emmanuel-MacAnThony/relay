package infra

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	ctx     context.Context
	queries *Queries
}

func NewRepo(ctx context.Context, pool *pgxpool.Pool) *Repo {
	return &Repo{ctx: ctx, queries: New(pool)}
}

func (r *Repo) Save(req domain.Request) error {
	headers, err := json.Marshal(req.Headers)
	if err != nil {
		return err
	}
	queryParams, err := json.Marshal(req.QueryParams)
	if err != nil {
		return err
	}

	return r.queries.SaveRequest(r.ctx, SaveRequestParams{
		ID:          req.ID,
		Slug:        req.Slug,
		Method:      req.Method,
		Path:        req.Path,
		Headers:     headers,
		QueryParams: queryParams,
		Body:        req.Body,
		SourceIp:    req.SourceIP,
		ReceivedAt:  pgtype.Timestamptz{Time: req.ReceivedAt, Valid: true},
		BodySize:    int32(req.BodySize),
	})
}

func (r *Repo) SaveDeliveryAttempt(d domain.DeliveryAttempt) error {
	responseHeaders, err := json.Marshal(d.ResponseHeaders)
	if err != nil {
		responseHeaders = []byte("{}")
	}
	return r.queries.SaveDeliveryAttempt(r.ctx, SaveDeliveryAttemptParams{
		ID:              d.ID,
		RequestID:       d.RequestID,
		Delivered:       d.Delivered,
		StatusCode:      int32(d.StatusCode),
		Error:           d.Error,
		IsReplay:        d.IsReplay,
		AttemptedAt:     pgtype.Timestamptz{Time: d.AttemptedAt, Valid: true},
		ResponseBody:    d.ResponseBody,
		ResponseHeaders: responseHeaders,
	})
}

func (r *Repo) List(slug string) ([]domain.Request, error) {
	rows, err := r.queries.ListRequests(r.ctx, slug)
	if err != nil {
		return nil, err
	}

	requests := make([]domain.Request, len(rows))
	for i, row := range rows {
		var headers map[string][]string
		var queryParams map[string][]string
		_ = json.Unmarshal(row.Headers, &headers)
		_ = json.Unmarshal(row.QueryParams, &queryParams)

		requests[i] = domain.Request{
			ID:          row.ID,
			Slug:        row.Slug,
			Method:      row.Method,
			Path:        row.Path,
			Headers:     headers,
			QueryParams: queryParams,
			Body:        row.Body,
			SourceIP:    row.SourceIp,
			ReceivedAt:  row.ReceivedAt.Time,
			BodySize:    int(row.BodySize),
		}
	}
	return requests, nil
}

func (r *Repo) ListLatestAttemptsBySlug(slug string) ([]domain.DeliveryAttempt, error) {
	rows, err := r.queries.ListLatestAttemptsBySlug(r.ctx, slug)
	if err != nil {
		return nil, err
	}
	attempts := make([]domain.DeliveryAttempt, len(rows))
	for i, row := range rows {
		var responseHeaders map[string][]string
		_ = json.Unmarshal(row.ResponseHeaders, &responseHeaders)
		attempts[i] = domain.DeliveryAttempt{
			ID:              row.ID,
			RequestID:       row.RequestID,
			Delivered:       row.Delivered,
			StatusCode:      int(row.StatusCode),
			Error:           row.Error,
			IsReplay:        row.IsReplay,
			AttemptedAt:     row.AttemptedAt.Time,
			ResponseBody:    row.ResponseBody,
			ResponseHeaders: responseHeaders,
		}
	}
	return attempts, nil
}

func (r *Repo) Get(id string) (domain.Request, error) {
	row, err := r.queries.GetRequest(r.ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Request{}, domain.ErrNotFound
		}
		return domain.Request{}, err
	}

	var headers map[string][]string
	var queryParams map[string][]string
	_ = json.Unmarshal(row.Headers, &headers)
	_ = json.Unmarshal(row.QueryParams, &queryParams)

	return domain.Request{
		ID:          row.ID,
		Slug:        row.Slug,
		Method:      row.Method,
		Path:        row.Path,
		Headers:     headers,
		QueryParams: queryParams,
		Body:        row.Body,
		SourceIP:    row.SourceIp,
		ReceivedAt:  row.ReceivedAt.Time,
		BodySize:    int(row.BodySize),
	}, nil
}
