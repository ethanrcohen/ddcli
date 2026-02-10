package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchLogs_Success(t *testing.T) {
	expectedResp := LogsListResponse{
		Data: []LogEntry{
			{
				ID:   "log-1",
				Type: "log",
				Attributes: LogAttributes{
					Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
					Status:    "error",
					Service:   "payment",
					Host:      "web-1",
					Message:   "payment failed: insufficient funds",
					Tags:      []string{"env:prod"},
				},
			},
			{
				ID:   "log-2",
				Type: "log",
				Attributes: LogAttributes{
					Timestamp: time.Date(2025, 1, 15, 10, 29, 0, 0, time.UTC),
					Status:    "error",
					Service:   "payment",
					Host:      "web-2",
					Message:   "payment timeout after 30s",
					Tags:      []string{"env:prod"},
				},
			},
		},
		Meta: LogsListMeta{
			Page:    LogsListPage{After: "cursor-abc"},
			Status:  "done",
			Elapsed: 42,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v2/logs/events/search", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-api-key", r.Header.Get("DD-API-KEY"))
		assert.Equal(t, "test-app-key", r.Header.Get("DD-APPLICATION-KEY"))

		// Verify request body
		var body searchLogsBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "service:payment status:error", body.Filter.Query)
		assert.Equal(t, "-timestamp", body.Sort)
		assert.Equal(t, 50, body.Page.Limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResp)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-api-key", "test-app-key")
	resp, err := client.SearchLogs(context.Background(), SearchLogsParams{
		Query: "service:payment status:error",
		From:  time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		To:    time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
		Limit: 50,
	})

	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "log-1", resp.Data[0].ID)
	assert.Equal(t, "payment failed: insufficient funds", resp.Data[0].Attributes.Message)
	assert.Equal(t, "cursor-abc", resp.Meta.Page.After)
}

func TestSearchLogs_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string][]string{
			"errors": {"Forbidden: invalid API key"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad-key", "bad-app-key")
	_, err := client.SearchLogs(context.Background(), SearchLogsParams{
		Query: "*",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
	})

	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.StatusCode)
	assert.Contains(t, apiErr.Error(), "Forbidden")
}

func TestSearchLogs_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string][]string{
			"errors": {"rate limited"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key", "app-key")
	_, err := client.SearchLogs(context.Background(), SearchLogsParams{
		Query: "*",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
	})

	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
}

func TestAggregateLogs_Success(t *testing.T) {
	expectedResp := LogsAggregateResponse{
		Data: LogsAggregateData{
			Buckets: []AggregateBucket{
				{
					By:       map[string]string{"service": "payment"},
					Computes: map[string]interface{}{"c0": float64(142)},
				},
				{
					By:       map[string]string{"service": "web-store"},
					Computes: map[string]interface{}{"c0": float64(89)},
				},
			},
		},
		Meta: LogsAggregateMeta{
			Status:  "done",
			Elapsed: 35,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v2/logs/analytics/aggregate", r.URL.Path)

		var body aggregateLogsBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "status:error", body.Filter.Query)
		assert.Len(t, body.Compute, 1)
		assert.Equal(t, "count", body.Compute[0].Aggregation)
		assert.Len(t, body.GroupBy, 1)
		assert.Equal(t, "service", body.GroupBy[0].Facet)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResp)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-api-key", "test-app-key")
	resp, err := client.AggregateLogs(context.Background(), AggregateLogsParams{
		Query: "status:error",
		From:  time.Date(2025, 1, 14, 12, 0, 0, 0, time.UTC),
		To:    time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		Compute: []AggregateCompute{
			{Aggregation: "count", Type: "total"},
		},
		GroupBy: []AggregateGroupBy{
			{Facet: "service", Limit: 10},
		},
	})

	require.NoError(t, err)
	assert.Len(t, resp.Data.Buckets, 2)
	assert.Equal(t, "payment", resp.Data.Buckets[0].By["service"])
}

func TestSearchLogs_LimitClamping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchLogsBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		// Should be clamped to 1000
		assert.Equal(t, 1000, body.Page.Limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LogsListResponse{})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key", "app-key")
	_, err := client.SearchLogs(context.Background(), SearchLogsParams{
		Query: "*",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
		Limit: 5000, // exceeds max
	})
	require.NoError(t, err)
}

func TestSearchLogs_DefaultSort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchLogsBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "-timestamp", body.Sort)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LogsListResponse{})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key", "app-key")
	_, err := client.SearchLogs(context.Background(), SearchLogsParams{
		Query: "*",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
	})
	require.NoError(t, err)
}
