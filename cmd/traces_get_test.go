package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethanrcohen/ddcli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSpansAPI struct {
	searchResp  *api.SpansListResponse
	searchErr   error
	searchCalls []api.SearchSpansParams
}

func (m *mockSpansAPI) SearchSpans(_ context.Context, params api.SearchSpansParams) (*api.SpansListResponse, error) {
	m.searchCalls = append(m.searchCalls, params)
	return m.searchResp, m.searchErr
}

func TestTracesGet_JSONOutput(t *testing.T) {
	mock := &mockSpansAPI{
		searchResp: &api.SpansListResponse{
			Data: []api.SpanEntry{
				{
					ID:   "span-1",
					Type: "spans",
					Attributes: api.SpanAttributes{
						StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
						TraceID:        "abc123",
						SpanID:         "span-1",
						Service:        "web-store",
						ResourceName:   "GET /api/products",
						OperationName:  "http.request",
						Custom:         map[string]any{"duration": float64(5000000)},
					},
				},
				{
					ID:   "span-2",
					Type: "spans",
					Attributes: api.SpanAttributes{
						StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 100000, time.UTC),
						TraceID:        "abc123",
						SpanID:         "span-2",
						ParentID:       "span-1",
						Service:        "product-db",
						ResourceName:   "SELECT products",
						OperationName:  "postgres.query",
						Custom:         map[string]any{"duration": float64(2000000)},
					},
				},
			},
			Meta: api.SpansListMeta{Status: "done"},
		},
	}

	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	cleanup := SetTracesGetDeps(mock, fixedNow)
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"traces", "get", "abc123", "--from", "1h", "--output", "json"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	require.Len(t, mock.searchCalls, 1)
	assert.Equal(t, "trace_id:abc123", mock.searchCalls[0].Query)
	assert.Equal(t, "timestamp", mock.searchCalls[0].Sort)
	assert.Equal(t, fixedNow.Add(-1*time.Hour), mock.searchCalls[0].From)

	var result api.SpansListResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result.Data, 2)
	assert.Equal(t, "web-store", result.Data[0].Attributes.Service)
}

func TestTracesGet_TableOutput(t *testing.T) {
	mock := &mockSpansAPI{
		searchResp: &api.SpansListResponse{
			Data: []api.SpanEntry{
				{
					ID:   "span-1",
					Type: "spans",
					Attributes: api.SpanAttributes{
						StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
						TraceID:        "abc123",
						SpanID:         "span-1",
						Service:        "web-store",
						ResourceName:   "GET /api/products",
						OperationName:  "http.request",
						Custom:         map[string]any{"duration": float64(5000000)},
					},
				},
				{
					ID:   "span-2",
					Type: "spans",
					Attributes: api.SpanAttributes{
						StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 100000, time.UTC),
						TraceID:        "abc123",
						SpanID:         "span-2",
						ParentID:       "span-1",
						Service:        "product-db",
						ResourceName:   "SELECT products",
						OperationName:  "postgres.query",
						Custom:         map[string]any{"duration": float64(2000000)},
					},
				},
			},
		},
	}

	cleanup := SetTracesGetDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"traces", "get", "abc123", "--from", "1h", "--output", "table"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "SERVICE")
	assert.Contains(t, out, "RESOURCE")
	assert.Contains(t, out, "web-store")
	assert.Contains(t, out, "product-db")
	assert.Contains(t, out, "(2 spans)")
}

func TestTracesGet_MissingTraceID(t *testing.T) {
	mock := &mockSpansAPI{
		searchResp: &api.SpansListResponse{},
	}

	cleanup := SetTracesGetDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"traces", "get"})

	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestTracesGet_Pagination(t *testing.T) {
	mockAPI := &multiPageSpansAPI{
		pages: []api.SpansListResponse{
			{
				Data: []api.SpanEntry{
					{ID: "span-1", Type: "spans", Attributes: api.SpanAttributes{SpanID: "span-1", Service: "svc-a"}},
				},
				Meta: api.SpansListMeta{Page: api.SpansListPage{After: "cursor-2"}},
			},
			{
				Data: []api.SpanEntry{
					{ID: "span-2", Type: "spans", Attributes: api.SpanAttributes{SpanID: "span-2", Service: "svc-b"}},
				},
				Meta: api.SpansListMeta{Page: api.SpansListPage{After: ""}},
			},
		},
	}

	cleanup := SetTracesGetDeps(mockAPI, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"traces", "get", "abc123", "--from", "1h", "--output", "json"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	// Should have made 2 API calls
	require.Len(t, mockAPI.calls, 2)
	assert.Empty(t, mockAPI.calls[0].Cursor)
	assert.Equal(t, "cursor-2", mockAPI.calls[1].Cursor)

	// Output should contain both spans
	var result api.SpansListResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result.Data, 2)
}

func TestTracesGet_EmptyResults(t *testing.T) {
	mock := &mockSpansAPI{
		searchResp: &api.SpansListResponse{
			Data: []api.SpanEntry{},
		},
	}

	cleanup := SetTracesGetDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"traces", "get", "nonexistent", "--from", "1h", "--output", "table"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "(no spans found)")
}

// multiPageSpansAPI is a mock that returns different pages sequentially.
type multiPageSpansAPI struct {
	pages []api.SpansListResponse
	calls []api.SearchSpansParams
	idx   int
}

func (m *multiPageSpansAPI) SearchSpans(_ context.Context, params api.SearchSpansParams) (*api.SpansListResponse, error) {
	m.calls = append(m.calls, params)
	if m.idx >= len(m.pages) {
		return &api.SpansListResponse{}, nil
	}
	resp := m.pages[m.idx]
	m.idx++
	return &resp, nil
}
