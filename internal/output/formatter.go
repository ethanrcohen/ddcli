package output

import (
	"fmt"
	"io"

	"github.com/ethanrcohen/ddcli/internal/api"
)

// Format represents an output format type.
type Format string

const (
	FormatJSON     Format = "json"
	FormatTable    Format = "table"
	FormatRaw      Format = "raw"
	FormatPerfetto Format = "perfetto"
)

// ParseFormat parses a string into a Format, returning an error for unknown formats.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatJSON, FormatTable, FormatRaw, FormatPerfetto:
		return Format(s), nil
	default:
		return "", fmt.Errorf("unknown output format %q (use json, table, raw, or perfetto)", s)
	}
}

// LogsFormatter formats log search results.
type LogsFormatter interface {
	FormatLogs(w io.Writer, resp *api.LogsListResponse) error
}

// AggregateFormatter formats log aggregation results.
type AggregateFormatter interface {
	FormatAggregate(w io.Writer, resp *api.LogsAggregateResponse) error
}

// NewLogsFormatter returns a LogsFormatter for the given output format.
func NewLogsFormatter(f Format) LogsFormatter {
	switch f {
	case FormatTable:
		return &TableFormatter{}
	case FormatRaw:
		return &RawFormatter{}
	default:
		return &JSONFormatter{}
	}
}

// NewAggregateFormatter returns an AggregateFormatter for the given output format.
func NewAggregateFormatter(f Format) AggregateFormatter {
	switch f {
	case FormatTable:
		return &TableFormatter{}
	case FormatRaw:
		return &RawFormatter{}
	default:
		return &JSONFormatter{}
	}
}

// SpansFormatter formats span search results.
type SpansFormatter interface {
	FormatSpans(w io.Writer, resp *api.SpansListResponse) error
}

// NewSpansFormatter returns a SpansFormatter for the given output format.
func NewSpansFormatter(f Format) SpansFormatter {
	switch f {
	case FormatTable:
		return &TableFormatter{}
	case FormatRaw:
		return &RawFormatter{}
	case FormatPerfetto:
		return &PerfettoFormatter{}
	default:
		return &JSONFormatter{}
	}
}
