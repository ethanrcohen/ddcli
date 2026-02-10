package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ethanrcohen/ddcli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleLogsResponse() *api.LogsListResponse {
	return &api.LogsListResponse{
		Data: []api.LogEntry{
			{
				ID:   "log-1",
				Type: "log",
				Attributes: api.LogAttributes{
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
				Attributes: api.LogAttributes{
					Timestamp: time.Date(2025, 1, 15, 10, 29, 0, 0, time.UTC),
					Status:    "info",
					Service:   "web-store",
					Host:      "web-2",
					Message:   "request completed in 250ms",
					Tags:      []string{"env:prod"},
				},
			},
		},
	}
}

func sampleAggregateResponse() *api.LogsAggregateResponse {
	return &api.LogsAggregateResponse{
		Data: api.LogsAggregateData{
			Buckets: []api.AggregateBucket{
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
	}
}

func TestJSONFormatter_Logs(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}
	err := f.FormatLogs(&buf, sampleLogsResponse())
	require.NoError(t, err)

	// Should be valid JSON
	var result api.LogsListResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result.Data, 2)
	assert.Equal(t, "log-1", result.Data[0].ID)
}

func TestJSONFormatter_Aggregate(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}
	err := f.FormatAggregate(&buf, sampleAggregateResponse())
	require.NoError(t, err)

	var result api.LogsAggregateResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result.Data.Buckets, 2)
}

func TestTableFormatter_Logs(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	err := f.FormatLogs(&buf, sampleLogsResponse())
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "TIMESTAMP")
	assert.Contains(t, out, "STATUS")
	assert.Contains(t, out, "SERVICE")
	assert.Contains(t, out, "payment")
	assert.Contains(t, out, "web-store")
	assert.Contains(t, out, "payment failed: insufficient funds")
	assert.Contains(t, out, "(2 results)")
}

func TestTableFormatter_Aggregate(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	err := f.FormatAggregate(&buf, sampleAggregateResponse())
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "service")
	assert.Contains(t, out, "payment")
	assert.Contains(t, out, "web-store")
}

func TestRawFormatter_Logs(t *testing.T) {
	var buf bytes.Buffer
	f := &RawFormatter{}
	err := f.FormatLogs(&buf, sampleLogsResponse())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)
	assert.Equal(t, "payment failed: insufficient funds", lines[0])
	assert.Equal(t, "request completed in 250ms", lines[1])
}

func TestRawFormatter_Aggregate(t *testing.T) {
	var buf bytes.Buffer
	f := &RawFormatter{}
	err := f.FormatAggregate(&buf, sampleAggregateResponse())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)
	// Each line should be valid JSON
	for _, line := range lines {
		var bucket api.AggregateBucket
		require.NoError(t, json.Unmarshal([]byte(line), &bucket))
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    Format
		wantErr bool
	}{
		{"json", FormatJSON, false},
		{"table", FormatTable, false},
		{"raw", FormatRaw, false},
		{"xml", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTableFormatter_Logs_Truncation(t *testing.T) {
	resp := &api.LogsListResponse{
		Data: []api.LogEntry{
			{
				ID: "log-1",
				Attributes: api.LogAttributes{
					Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
					Status:    "info",
					Service:   "test",
					Host:      "host-1",
					Message:   strings.Repeat("a", 200), // longer than 100 char truncation
				},
			},
		},
	}

	var buf bytes.Buffer
	f := &TableFormatter{}
	err := f.FormatLogs(&buf, resp)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "...")
	// Message in table should not be full 200 chars
	assert.Less(t, len(strings.Split(out, "\n")[2]), 200)
}
