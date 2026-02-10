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
		{"perfetto", FormatPerfetto, false},
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

// --- Spans formatter tests ---

func sampleSpansResponse() *api.SpansListResponse {
	return &api.SpansListResponse{
		Data: []api.SpanEntry{
			{
				ID:   "span-1",
				Type: "spans",
				Attributes: api.SpanAttributes{
					StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
					EndTimestamp:   time.Date(2025, 1, 15, 10, 30, 0, 5000000, time.UTC),
					TraceID:        "abc123",
					SpanID:         "span-1",
					Service:        "web-store",
					ResourceName:   "GET /api/products",
					OperationName:  "http.request",
					Custom:         map[string]any{"duration": float64(5000000)}, // 5ms
					Status:         "ok",
				},
			},
			{
				ID:   "span-2",
				Type: "spans",
				Attributes: api.SpanAttributes{
					StartTimestamp: time.Date(2025, 1, 15, 10, 30, 0, 500000, time.UTC),
					EndTimestamp:   time.Date(2025, 1, 15, 10, 30, 0, 2500000, time.UTC),
					TraceID:        "abc123",
					SpanID:         "span-2",
					ParentID:       "span-1",
					Service:        "product-db",
					ResourceName:   "SELECT products",
					OperationName:  "postgres.query",
					Custom:         map[string]any{"duration": float64(2000000)}, // 2ms
					Status:         "ok",
				},
			},
		},
	}
}

func TestJSONFormatter_Spans(t *testing.T) {
	var buf bytes.Buffer
	f := &JSONFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	var result api.SpansListResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result.Data, 2)
	assert.Equal(t, "span-1", result.Data[0].ID)
}

func TestTableFormatter_Spans(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "SERVICE")
	assert.Contains(t, out, "RESOURCE")
	assert.Contains(t, out, "DURATION")
	assert.Contains(t, out, "web-store")
	assert.Contains(t, out, "product-db")
	assert.Contains(t, out, "(2 spans)")
}

func TestTableFormatter_Spans_Tree(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	out := buf.String()
	// Child span (product-db) should be indented under parent (web-store)
	lines := strings.Split(out, "\n")
	var webStoreLine, productDBLine string
	for _, line := range lines {
		if strings.Contains(line, "web-store") {
			webStoreLine = line
		}
		if strings.Contains(line, "product-db") {
			productDBLine = line
		}
	}
	require.NotEmpty(t, webStoreLine, "should contain web-store line")
	require.NotEmpty(t, productDBLine, "should contain product-db line")
	// product-db line should start with more whitespace (indented as child)
	assert.True(t, len(productDBLine)-len(strings.TrimLeft(productDBLine, " ")) >
		len(webStoreLine)-len(strings.TrimLeft(webStoreLine, " ")),
		"child span should be more indented than parent")
}

func TestTableFormatter_Spans_Empty(t *testing.T) {
	var buf bytes.Buffer
	f := &TableFormatter{}
	err := f.FormatSpans(&buf, &api.SpansListResponse{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "(no spans found)")
}

func TestRawFormatter_Spans(t *testing.T) {
	var buf bytes.Buffer
	f := &RawFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)
	// Each line should be valid JSON
	for _, line := range lines {
		var entry api.SpanEntry
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
	}
}

// --- Perfetto formatter tests ---

func TestPerfettoFormatter_ValidChromeTraceFormat(t *testing.T) {
	var buf bytes.Buffer
	f := &PerfettoFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	var events []traceEvent
	require.NoError(t, json.Unmarshal(buf.Bytes(), &events))

	// 2 metadata events (one per service) + 2 span events
	assert.Len(t, events, 4)
}

func TestPerfettoFormatter_MetadataEvents(t *testing.T) {
	var buf bytes.Buffer
	f := &PerfettoFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	var events []traceEvent
	require.NoError(t, json.Unmarshal(buf.Bytes(), &events))

	// Collect metadata events
	var metaEvents []traceEvent
	for _, ev := range events {
		if ev.Ph == "M" {
			metaEvents = append(metaEvents, ev)
		}
	}
	assert.Len(t, metaEvents, 2)

	// Each should have process_name and a string name arg
	serviceNames := make(map[string]bool)
	for _, ev := range metaEvents {
		assert.Equal(t, "process_name", ev.Name)
		assert.NotZero(t, ev.Pid)
		name, ok := ev.Args["name"].(string)
		require.True(t, ok)
		serviceNames[name] = true
	}
	assert.True(t, serviceNames["web-store"])
	assert.True(t, serviceNames["product-db"])
}

func TestPerfettoFormatter_SpanEvents(t *testing.T) {
	var buf bytes.Buffer
	f := &PerfettoFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	var events []traceEvent
	require.NoError(t, json.Unmarshal(buf.Bytes(), &events))

	// Collect span events
	var spanEvents []traceEvent
	for _, ev := range events {
		if ev.Ph == "X" {
			spanEvents = append(spanEvents, ev)
		}
	}
	require.Len(t, spanEvents, 2)

	// First span: web-store root
	root := spanEvents[0]
	assert.Equal(t, "GET /api/products", root.Name)
	assert.Equal(t, "http.request", root.Cat)
	assert.Equal(t, int64(5000), root.Dur) // 5ms = 5000us
	assert.NotZero(t, root.Ts)
	assert.Equal(t, "span-1", root.Args["span_id"])
	assert.Equal(t, "abc123", root.Args["trace_id"])

	// Second span: product-db child
	child := spanEvents[1]
	assert.Equal(t, "SELECT products", child.Name)
	assert.Equal(t, int64(2000), child.Dur) // 2ms = 2000us
	assert.Equal(t, "span-1", child.Args["parent_id"])
}

func TestPerfettoFormatter_NumericPids(t *testing.T) {
	var buf bytes.Buffer
	f := &PerfettoFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	var events []traceEvent
	require.NoError(t, json.Unmarshal(buf.Bytes(), &events))

	// All pids must be positive integers (Chrome Trace Format requirement)
	for _, ev := range events {
		assert.Greater(t, ev.Pid, 0, "pid must be a positive integer")
	}

	// Spans from the same service should share a pid
	var spanEvents []traceEvent
	for _, ev := range events {
		if ev.Ph == "X" {
			spanEvents = append(spanEvents, ev)
		}
	}
	// web-store and product-db should have different pids
	assert.NotEqual(t, spanEvents[0].Pid, spanEvents[1].Pid)
}

func TestPerfettoFormatter_Empty(t *testing.T) {
	var buf bytes.Buffer
	f := &PerfettoFormatter{}
	err := f.FormatSpans(&buf, &api.SpansListResponse{})
	require.NoError(t, err)

	var events []traceEvent
	require.NoError(t, json.Unmarshal(buf.Bytes(), &events))
	assert.Empty(t, events)
}

func TestPerfettoFormatter_ParentIDOmittedForRoot(t *testing.T) {
	var buf bytes.Buffer
	f := &PerfettoFormatter{}
	err := f.FormatSpans(&buf, sampleSpansResponse())
	require.NoError(t, err)

	var events []traceEvent
	require.NoError(t, json.Unmarshal(buf.Bytes(), &events))

	for _, ev := range events {
		if ev.Ph == "X" && ev.Name == "GET /api/products" {
			// Root span should not have parent_id in args
			_, hasParent := ev.Args["parent_id"]
			assert.False(t, hasParent, "root span should not have parent_id")
		}
		if ev.Ph == "X" && ev.Name == "SELECT products" {
			// Child span should have parent_id
			_, hasParent := ev.Args["parent_id"]
			assert.True(t, hasParent, "child span should have parent_id")
		}
	}
}
