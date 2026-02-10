package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SpansAPI defines the interface for Datadog Spans API operations.
type SpansAPI interface {
	SearchSpans(ctx context.Context, params SearchSpansParams) (*SpansListResponse, error)
}

// --- Request types ---

// SearchSpansParams are the parameters for a span search request.
type SearchSpansParams struct {
	Query  string
	From   time.Time
	To     time.Time
	Sort   string // "timestamp" (oldest first, default) or "-timestamp" (newest first)
	Limit  int
	Cursor string
}

// --- Response types ---

// SpansListResponse is the response from the search spans endpoint.
type SpansListResponse struct {
	Data []SpanEntry   `json:"data"`
	Meta SpansListMeta `json:"meta"`
}

type SpanEntry struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Attributes SpanAttributes `json:"attributes"`
}

type SpanAttributes struct {
	StartTimestamp time.Time      `json:"start_timestamp"`
	EndTimestamp   time.Time      `json:"end_timestamp"`
	TraceID        string         `json:"trace_id"`
	SpanID         string         `json:"span_id"`
	ParentID       string         `json:"parent_id"`
	Service        string         `json:"service"`
	ResourceName   string         `json:"resource_name"`
	OperationName  string         `json:"operation_name"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	Custom         map[string]any `json:"custom"`
	Tags           []string       `json:"tags"`
}

// Duration returns the span duration in nanoseconds from the custom attributes.
func (a *SpanAttributes) Duration() int64 {
	if a.Custom == nil {
		return 0
	}
	if d, ok := a.Custom["duration"]; ok {
		if f, ok := d.(float64); ok {
			return int64(f)
		}
	}
	return 0
}

type SpansListMeta struct {
	Page    SpansListPage `json:"page"`
	Status  string        `json:"status"`
	Elapsed int           `json:"elapsed"`
}

type SpansListPage struct {
	After string `json:"after"`
}

// --- API request body types (internal, match DD API schema) ---

type searchSpansBody struct {
	Data searchSpansData `json:"data"`
}

type searchSpansData struct {
	Type       string           `json:"type"`
	Attributes searchSpansAttrs `json:"attributes"`
}

type searchSpansAttrs struct {
	Filter searchSpansFilter `json:"filter"`
	Sort   string            `json:"sort,omitempty"`
	Page   searchSpansPage   `json:"page,omitempty"`
}

type searchSpansFilter struct {
	Query string `json:"query"`
	From  string `json:"from"`
	To    string `json:"to"`
}

type searchSpansPage struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// --- Implementation ---

func (c *Client) SearchSpans(ctx context.Context, params SearchSpansParams) (*SpansListResponse, error) {
	sort := params.Sort
	if sort == "" {
		sort = "timestamp"
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	body := searchSpansBody{
		Data: searchSpansData{
			Type: "search_request",
			Attributes: searchSpansAttrs{
				Filter: searchSpansFilter{
					Query: params.Query,
					From:  params.From.UTC().Format(time.RFC3339),
					To:    params.To.UTC().Format(time.RFC3339),
				},
				Sort: sort,
				Page: searchSpansPage{
					Limit:  limit,
					Cursor: params.Cursor,
				},
			},
		},
	}

	respBody, err := c.do(ctx, "POST", "/api/v2/spans/events/search", body)
	if err != nil {
		return nil, fmt.Errorf("searching spans: %w", err)
	}

	var result SpansListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding spans search response: %w", err)
	}
	return &result, nil
}
