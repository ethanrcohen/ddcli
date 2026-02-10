package timeutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRelativeOrAbsolute(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr string
	}{
		{
			name:  "empty string returns now",
			input: "",
			want:  now,
		},
		{
			name:  "now returns now",
			input: "now",
			want:  now,
		},
		// Relative durations
		{
			name:  "15 minutes ago",
			input: "15m",
			want:  now.Add(-15 * time.Minute),
		},
		{
			name:  "1 hour ago",
			input: "1h",
			want:  now.Add(-1 * time.Hour),
		},
		{
			name:  "24 hours ago",
			input: "24h",
			want:  now.Add(-24 * time.Hour),
		},
		{
			name:  "7 days ago",
			input: "7d",
			want:  now.Add(-7 * 24 * time.Hour),
		},
		{
			name:  "2 weeks ago",
			input: "2w",
			want:  now.Add(-14 * 24 * time.Hour),
		},
		{
			name:  "30 seconds ago",
			input: "30s",
			want:  now.Add(-30 * time.Second),
		},
		// Absolute timestamps
		{
			name:  "RFC3339",
			input: "2024-01-01T00:00:00Z",
			want:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "RFC3339 with timezone",
			input: "2024-06-15T08:30:00+05:00",
			want:  time.Date(2024, 6, 15, 8, 30, 0, 0, time.FixedZone("", 5*3600)),
		},
		{
			name:  "date only",
			input: "2024-01-01",
			want:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "datetime without timezone",
			input: "2024-01-01T15:30:00",
			want:  time.Date(2024, 1, 1, 15, 30, 0, 0, time.UTC),
		},
		// Errors
		{
			name:    "invalid string",
			input:   "foobar",
			wantErr: "cannot parse time",
		},
		{
			name:    "single character",
			input:   "x",
			wantErr: "cannot parse time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRelativeOrAbsolute(tt.input, now)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "expected %v, got %v", tt.want, got)
		})
	}
}
