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

func TestSearchSpans_Success(t *testing.T) {
	expectedResp := SpansListResponse{
		Data: []SpanEntry{
			{
				ID:   "span-1",
				Type: "spans",
				Attributes: SpanAttributes{
					StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
					TraceID:        "abc123",
					SpanID:         "span-1",
					ParentID:       "",
					Service:        "web-store",
					ResourceName:   "GET /api/products",
					OperationName:  "http.request",
					Custom:         map[string]any{"duration": float64(5000000)},
					Status:         "ok",
				},
			},
			{
				ID:   "span-2",
				Type: "spans",
				Attributes: SpanAttributes{
					StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 100000, time.UTC),
					TraceID:        "abc123",
					SpanID:         "span-2",
					ParentID:       "span-1",
					Service:        "product-db",
					ResourceName:   "SELECT products",
					OperationName:  "postgres.query",
					Custom:         map[string]any{"duration": float64(2000000)},
					Status:         "ok",
				},
			},
		},
		Meta: SpansListMeta{
			Page:    SpansListPage{After: "cursor-xyz"},
			Status:  "done",
			Elapsed: 15,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v2/spans/events/search", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-api-key", r.Header.Get("DD-API-KEY"))
		assert.Equal(t, "test-app-key", r.Header.Get("DD-APPLICATION-KEY"))

		var body searchSpansBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "search_request", body.Data.Type)
		assert.Equal(t, "trace_id:abc123", body.Data.Attributes.Filter.Query)
		assert.Equal(t, "timestamp", body.Data.Attributes.Sort)
		assert.Equal(t, 50, body.Data.Attributes.Page.Limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResp)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-api-key", "test-app-key")
	resp, err := client.SearchSpans(context.Background(), SearchSpansParams{
		Query: "trace_id:abc123",
		From:  time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		To:    time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
		Limit: 50,
	})

	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "span-1", resp.Data[0].ID)
	assert.Equal(t, "abc123", resp.Data[0].Attributes.TraceID)
	assert.Equal(t, "cursor-xyz", resp.Meta.Page.After)
}

func TestSearchSpans_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string][]string{
			"errors": {"Forbidden: invalid API key"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad-key", "bad-app-key")
	_, err := client.SearchSpans(context.Background(), SearchSpansParams{
		Query: "trace_id:abc123",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
	})

	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.StatusCode)
	assert.Contains(t, apiErr.Error(), "Forbidden")
}

func TestSearchSpans_LimitClamping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchSpansBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, 1000, body.Data.Attributes.Page.Limit)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SpansListResponse{})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key", "app-key")
	_, err := client.SearchSpans(context.Background(), SearchSpansParams{
		Query: "trace_id:abc123",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
		Limit: 5000,
	})
	require.NoError(t, err)
}

func TestSearchSpans_DefaultSort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchSpansBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "timestamp", body.Data.Attributes.Sort)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SpansListResponse{})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key", "app-key")
	_, err := client.SearchSpans(context.Background(), SearchSpansParams{
		Query: "trace_id:abc123",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
	})
	require.NoError(t, err)
}

func TestSearchSpans_CursorPagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchSpansBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			assert.Empty(t, body.Data.Attributes.Page.Cursor)
			json.NewEncoder(w).Encode(SpansListResponse{
				Data: []SpanEntry{{ID: "span-1", Type: "spans"}},
				Meta: SpansListMeta{Page: SpansListPage{After: "cursor-page2"}},
			})
		} else {
			assert.Equal(t, "cursor-page2", body.Data.Attributes.Page.Cursor)
			json.NewEncoder(w).Encode(SpansListResponse{
				Data: []SpanEntry{{ID: "span-2", Type: "spans"}},
				Meta: SpansListMeta{Page: SpansListPage{After: ""}},
			})
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key", "app-key")

	// First page
	resp, err := client.SearchSpans(context.Background(), SearchSpansParams{
		Query: "trace_id:abc123",
		From:  time.Now().Add(-1 * time.Hour),
		To:    time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, "cursor-page2", resp.Meta.Page.After)

	// Second page with cursor
	resp, err = client.SearchSpans(context.Background(), SearchSpansParams{
		Query:  "trace_id:abc123",
		From:   time.Now().Add(-1 * time.Hour),
		To:     time.Now(),
		Cursor: resp.Meta.Page.After,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Meta.Page.After)
	assert.Equal(t, 2, callCount)
}
