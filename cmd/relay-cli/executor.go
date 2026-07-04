package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Emmanuel-MacAnThony/relay/internal/request/domain"
	"github.com/google/uuid"
)

const maxResponseBodyBytes = 1 << 16 // 64 KB

type Executor struct {
	target string
	client *http.Client
}

func NewExecutor(target string) *Executor {
	return &Executor{
		target: target,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *Executor) Execute(ctx context.Context, req domain.Request) domain.DeliveryAttempt {
	attempt := domain.DeliveryAttempt{
		ID:          uuid.NewString(),
		RequestID:   req.ID,
		Slug:        req.Slug,
		AttemptedAt: time.Now().UTC(),
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, e.buildURL(req), bytes.NewReader(req.Body))
	if err != nil {
		attempt.Error = fmt.Sprintf("failed to build request: %s", err)
		return attempt
	}

	for k, vals := range req.Headers {
		if strings.EqualFold(k, "Host") {
			continue
		}
		for _, v := range vals {
			httpReq.Header.Add(k, v)
		}
	}

	resp, err := e.client.Do(httpReq)
	if err != nil {
		attempt.Error = fmt.Sprintf("request failed: %s", err)
		return attempt
	}
	defer resp.Body.Close()

	attempt.StatusCode = resp.StatusCode
	attempt.Delivered = resp.StatusCode >= 200 && resp.StatusCode < 300
	attempt.ResponseHeaders = map[string][]string(resp.Header)
	attempt.ResponseBody, _ = io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	return attempt
}

func (e *Executor) buildURL(req domain.Request) string {
	if len(req.QueryParams) == 0 {
		return e.target
	}
	return e.target + "?" + url.Values(req.QueryParams).Encode()
}
