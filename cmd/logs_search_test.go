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

// mockLogsAPI is a test double for the LogsAPI interface.
type mockLogsAPI struct {
	searchResp    *api.LogsListResponse
	searchErr     error
	searchCalls   []api.SearchLogsParams
	aggregateResp *api.LogsAggregateResponse
	aggregateErr  error
	aggregateCalls []api.AggregateLogsParams
}

func (m *mockLogsAPI) SearchLogs(_ context.Context, params api.SearchLogsParams) (*api.LogsListResponse, error) {
	m.searchCalls = append(m.searchCalls, params)
	return m.searchResp, m.searchErr
}

func (m *mockLogsAPI) AggregateLogs(_ context.Context, params api.AggregateLogsParams) (*api.LogsAggregateResponse, error) {
	m.aggregateCalls = append(m.aggregateCalls, params)
	return m.aggregateResp, m.aggregateErr
}

func TestLogsSearch_JSONOutput(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{
			Data: []api.LogEntry{
				{
					ID:   "log-1",
					Type: "log",
					Attributes: api.LogAttributes{
						Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
						Status:    "error",
						Service:   "payment",
						Host:      "web-1",
						Message:   "payment failed",
					},
				},
			},
			Meta: api.LogsListMeta{Status: "done"},
		},
	}

	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	cleanup := SetLogsSearchDeps(mock, fixedNow)
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"logs", "search", "service:payment status:error", "--from", "1h", "--output", "json"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	// Verify the mock was called with correct params
	require.Len(t, mock.searchCalls, 1)
	assert.Equal(t, "service:payment status:error", mock.searchCalls[0].Query)
	assert.Equal(t, fixedNow.Add(-1*time.Hour), mock.searchCalls[0].From)

	// Verify JSON output
	var result api.LogsListResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result.Data, 1)
	assert.Equal(t, "payment failed", result.Data[0].Attributes.Message)
}

func TestLogsSearch_TableOutput(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{
			Data: []api.LogEntry{
				{
					ID: "log-1",
					Attributes: api.LogAttributes{
						Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
						Status:    "error",
						Service:   "payment",
						Host:      "web-1",
						Message:   "payment failed",
					},
				},
			},
		},
	}

	cleanup := SetLogsSearchDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"logs", "search", "service:payment", "--from", "1h", "--output", "table"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "TIMESTAMP")
	assert.Contains(t, out, "payment failed")
	assert.Contains(t, out, "(1 results)")
}

func TestLogsSearch_RawOutput(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{
			Data: []api.LogEntry{
				{
					ID: "log-1",
					Attributes: api.LogAttributes{
						Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
						Message:   "first message",
					},
				},
				{
					ID: "log-2",
					Attributes: api.LogAttributes{
						Timestamp: time.Date(2025, 1, 15, 10, 29, 0, 0, time.UTC),
						Message:   "second message",
					},
				},
			},
		},
	}

	cleanup := SetLogsSearchDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"logs", "search", "*", "--from", "1h", "--output", "raw"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "first message")
	assert.Contains(t, out, "second message")
}

func TestLogsSearch_DefaultQuery(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{},
	}

	cleanup := SetLogsSearchDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"logs", "search", "--from", "1h"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	require.Len(t, mock.searchCalls, 1)
	assert.Equal(t, "*", mock.searchCalls[0].Query)
}

func TestLogsSearch_AbsoluteTimeRange(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{},
	}

	cleanup := SetLogsSearchDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"logs", "search", "service:web",
		"--from", "2025-01-01T00:00:00Z",
		"--to", "2025-01-02T00:00:00Z",
	})

	err := rootCmd.Execute()
	require.NoError(t, err)

	require.Len(t, mock.searchCalls, 1)
	assert.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), mock.searchCalls[0].From)
	assert.Equal(t, time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), mock.searchCalls[0].To)
}

func TestLogsSearch_ServiceFlag(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{},
	}

	cleanup := SetLogsSearchDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"logs", "search", "--service", "payment", "--from", "1h"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	require.Len(t, mock.searchCalls, 1)
	assert.Equal(t, "service:payment", mock.searchCalls[0].Query)
}

func TestLogsSearch_AllFilterFlags(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{},
	}

	cleanup := SetLogsSearchDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"logs", "search",
		"--service", "payment",
		"--env", "prod",
		"--host", "web-1",
		"--status", "error",
		"--from", "1h",
	})

	err := rootCmd.Execute()
	require.NoError(t, err)

	require.Len(t, mock.searchCalls, 1)
	assert.Equal(t, "service:payment env:prod host:web-1 status:error", mock.searchCalls[0].Query)
}

func TestLogsSearch_FlagsCombinedWithQuery(t *testing.T) {
	mock := &mockLogsAPI{
		searchResp: &api.LogsListResponse{},
	}

	cleanup := SetLogsSearchDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"logs", "search", "@duration:>5s",
		"--service", "payment",
		"--env", "prod",
		"--host", "",
		"--status", "",
		"--from", "1h",
	})

	err := rootCmd.Execute()
	require.NoError(t, err)

	require.Len(t, mock.searchCalls, 1)
	assert.Equal(t, "service:payment env:prod @duration:>5s", mock.searchCalls[0].Query)
}
