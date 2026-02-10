package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethanrcohen/ddcli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsAggregate_CountByService(t *testing.T) {
	mock := &mockLogsAPI{
		aggregateResp: &api.LogsAggregateResponse{
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
		},
	}

	cleanup := SetLogsAggregateDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"logs", "aggregate", "status:error",
		"--compute", "count",
		"--group-by", "service",
		"--from", "24h",
	})

	err := rootCmd.Execute()
	require.NoError(t, err)

	// Verify mock was called correctly
	require.Len(t, mock.aggregateCalls, 1)
	call := mock.aggregateCalls[0]
	assert.Equal(t, "status:error", call.Query)
	assert.Len(t, call.Compute, 1)
	assert.Equal(t, "count", call.Compute[0].Aggregation)
	assert.Len(t, call.GroupBy, 1)
	assert.Equal(t, "service", call.GroupBy[0].Facet)

	// Verify output is valid JSON
	var result api.LogsAggregateResponse
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Len(t, result.Data.Buckets, 2)
}

func TestLogsAggregate_AvgMetric(t *testing.T) {
	mock := &mockLogsAPI{
		aggregateResp: &api.LogsAggregateResponse{
			Data: api.LogsAggregateData{
				Buckets: []api.AggregateBucket{
					{
						By:       map[string]string{"service": "api"},
						Computes: map[string]interface{}{"c0": float64(1.5)},
					},
				},
			},
		},
	}

	cleanup := SetLogsAggregateDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"logs", "aggregate", "*",
		"--compute", "avg:@duration",
		"--group-by", "service",
		"--from", "1h",
	})

	err := rootCmd.Execute()
	require.NoError(t, err)

	require.Len(t, mock.aggregateCalls, 1)
	assert.Equal(t, "avg", mock.aggregateCalls[0].Compute[0].Aggregation)
	assert.Equal(t, "@duration", mock.aggregateCalls[0].Compute[0].Metric)
}

func TestLogsAggregate_TableOutput(t *testing.T) {
	mock := &mockLogsAPI{
		aggregateResp: &api.LogsAggregateResponse{
			Data: api.LogsAggregateData{
				Buckets: []api.AggregateBucket{
					{
						By:       map[string]string{"service": "payment"},
						Computes: map[string]interface{}{"c0": float64(142)},
					},
				},
			},
		},
	}

	cleanup := SetLogsAggregateDeps(mock, time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	defer cleanup()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{
		"logs", "aggregate", "status:error",
		"--compute", "count",
		"--group-by", "service",
		"--from", "1h",
		"--output", "table",
	})

	err := rootCmd.Execute()
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "service")
	assert.Contains(t, out, "payment")
}

func TestParseCompute(t *testing.T) {
	tests := []struct {
		input       string
		wantAgg     string
		wantMetric  string
		wantErr     bool
	}{
		{"count", "count", "", false},
		{"avg:@duration", "avg", "@duration", false},
		{"sum:@bytes", "sum", "@bytes", false},
		{"min:@latency", "min", "@latency", false},
		{"max:@latency", "max", "@latency", false},
		{"avg:", "", "", true},     // missing metric
		{"unknown", "", "", true},  // unknown aggregation
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseCompute(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAgg, got.Aggregation)
			assert.Equal(t, tt.wantMetric, got.Metric)
		})
	}
}
