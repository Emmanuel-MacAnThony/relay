package domain

import "time"

type Request struct {
	ID          string              `json:"id"`
	Slug        string              `json:"slug"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Headers     map[string][]string `json:"headers"`
	QueryParams map[string][]string `json:"query_params"`
	Body        []byte              `json:"body"`
	SourceIP    string              `json:"source_ip"`
	ReceivedAt  time.Time           `json:"received_at"`
	BodySize    int                 `json:"body_size"`
}

type DeliveryAttempt struct {
	ID              string              `json:"id"`
	RequestID       string              `json:"request_id"`
	Slug            string              `json:"slug"`
	Delivered       bool                `json:"delivered"`
	StatusCode      int                 `json:"status_code"`
	Error           string              `json:"error"`
	IsReplay        bool                `json:"is_replay"`
	AttemptedAt     time.Time           `json:"attempted_at"`
	ResponseBody    []byte              `json:"response_body,omitempty"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
}
