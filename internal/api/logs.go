package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// --- Request types ---

// SearchLogsParams are the parameters for a log search request.
type SearchLogsParams struct {
	Query   string
	From    time.Time
	To      time.Time
	Sort    string // "-timestamp" (newest first, default) or "timestamp" (oldest first)
	Limit   int
	Cursor  string
	Indexes []string
}

// AggregateLogsParams are the parameters for a log aggregation request.
type AggregateLogsParams struct {
	Query       string
	From        time.Time
	To          time.Time
	Compute     []AggregateCompute
	GroupBy     []AggregateGroupBy
	Type        string // "total" or "timeseries"
}

type AggregateCompute struct {
	Aggregation string `json:"aggregation"`          // count, avg, sum, min, max, pct
	Metric      string `json:"metric,omitempty"`      // e.g. "@duration"
	Type        string `json:"type"`                  // "total" or "timeseries"
}

type AggregateGroupBy struct {
	Facet string              `json:"facet"`
	Limit int                 `json:"limit,omitempty"`
	Sort  *AggregateGroupSort `json:"sort,omitempty"`
}

type AggregateGroupSort struct {
	Aggregation string `json:"aggregation"`
	Order       string `json:"order"` // "asc" or "desc"
}

// --- Response types ---

// LogsListResponse is the response from the search/list logs endpoint.
type LogsListResponse struct {
	Data []LogEntry     `json:"data"`
	Meta LogsListMeta   `json:"meta"`
}

type LogEntry struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes LogAttributes   `json:"attributes"`
}

type LogAttributes struct {
	Timestamp  time.Time              `json:"timestamp"`
	Status     string                 `json:"status"`
	Service    string                 `json:"service"`
	Host       string                 `json:"host"`
	Message    string                 `json:"message"`
	Attributes map[string]any         `json:"attributes"`
	Tags       []string               `json:"tags"`
}

type LogsListMeta struct {
	Page    LogsListPage `json:"page"`
	Status  string       `json:"status"`
	Elapsed int          `json:"elapsed"`
}

type LogsListPage struct {
	After string `json:"after"`
}

// LogsAggregateResponse is the response from the aggregate logs endpoint.
type LogsAggregateResponse struct {
	Data LogsAggregateData `json:"data"`
	Meta LogsAggregateMeta `json:"meta"`
}

type LogsAggregateData struct {
	Buckets []AggregateBucket `json:"buckets"`
}

type AggregateBucket struct {
	By       map[string]string      `json:"by"`
	Computes map[string]interface{} `json:"computes"`
}

type LogsAggregateMeta struct {
	Status  string `json:"status"`
	Elapsed int    `json:"elapsed"`
}

// --- API request body types (internal, match DD API schema) ---

type searchLogsBody struct {
	Filter searchFilter `json:"filter"`
	Sort   string       `json:"sort,omitempty"`
	Page   searchPage   `json:"page,omitempty"`
}

type searchFilter struct {
	Query   string   `json:"query"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Indexes []string `json:"indexes,omitempty"`
}

type searchPage struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type aggregateLogsBody struct {
	Compute []AggregateCompute  `json:"compute"`
	Filter  searchFilter        `json:"filter"`
	GroupBy []AggregateGroupBy  `json:"group_by,omitempty"`
}

// --- Implementation ---

func (c *Client) SearchLogs(ctx context.Context, params SearchLogsParams) (*LogsListResponse, error) {
	sort := params.Sort
	if sort == "" {
		sort = "-timestamp"
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	body := searchLogsBody{
		Filter: searchFilter{
			Query:   params.Query,
			From:    params.From.UTC().Format(time.RFC3339),
			To:      params.To.UTC().Format(time.RFC3339),
			Indexes: params.Indexes,
		},
		Sort: sort,
		Page: searchPage{
			Limit:  limit,
			Cursor: params.Cursor,
		},
	}

	respBody, err := c.do(ctx, "POST", "/api/v2/logs/events/search", body)
	if err != nil {
		return nil, fmt.Errorf("searching logs: %w", err)
	}

	var result LogsListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}
	return &result, nil
}

func (c *Client) AggregateLogs(ctx context.Context, params AggregateLogsParams) (*LogsAggregateResponse, error) {
	body := aggregateLogsBody{
		Compute: params.Compute,
		Filter: searchFilter{
			Query: params.Query,
			From:  params.From.UTC().Format(time.RFC3339),
			To:    params.To.UTC().Format(time.RFC3339),
		},
		GroupBy: params.GroupBy,
	}

	respBody, err := c.do(ctx, "POST", "/api/v2/logs/analytics/aggregate", body)
	if err != nil {
		return nil, fmt.Errorf("aggregating logs: %w", err)
	}

	var result LogsAggregateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding aggregate response: %w", err)
	}
	return &result, nil
}
