package output

import (
	"encoding/json"
	"io"

	"github.com/ethanrcohen/ddcli/internal/api"
)

// PerfettoFormatter outputs spans as Chrome Trace Event Format JSON.
// Open the output in https://ui.perfetto.dev, chrome://tracing, or speedscope.
type PerfettoFormatter struct{}

// traceEvent is a Chrome Trace Event Format event.
// See: https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU
type traceEvent struct {
	Name string         `json:"name"`
	Cat  string         `json:"cat,omitempty"`
	Ph   string         `json:"ph"`
	Ts   int64          `json:"ts,omitempty"`  // microseconds
	Dur  int64          `json:"dur,omitempty"` // microseconds
	Pid  int            `json:"pid"`
	Tid  int            `json:"tid"`
	Args map[string]any `json:"args,omitempty"`
}

func (f *PerfettoFormatter) FormatSpans(w io.Writer, resp *api.SpansListResponse) error {
	// Assign each unique service a stable numeric pid
	servicePIDs := make(map[string]int)
	nextPID := 1
	for _, span := range resp.Data {
		svc := span.Attributes.Service
		if _, ok := servicePIDs[svc]; !ok {
			servicePIDs[svc] = nextPID
			nextPID++
		}
	}

	events := make([]traceEvent, 0, len(resp.Data)+len(servicePIDs))

	// Metadata events to label processes with service names
	for svc, pid := range servicePIDs {
		events = append(events, traceEvent{
			Name: "process_name",
			Ph:   "M",
			Pid:  pid,
			Args: map[string]any{"name": svc},
		})
	}

	for _, span := range resp.Data {
		a := span.Attributes
		durUs := a.Duration() / 1000

		ev := traceEvent{
			Name: a.ResourceName,
			Cat:  a.OperationName,
			Ph:   "X",
			Ts:   a.StartTimestamp.UnixMicro(),
			Dur:  durUs,
			Pid:  servicePIDs[a.Service],
			Tid:  servicePIDs[a.Service],
			Args: map[string]any{
				"span_id":  a.SpanID,
				"trace_id": a.TraceID,
				"status":   a.Status,
			},
		}

		if a.ParentID != "" {
			ev.Args["parent_id"] = a.ParentID
		}

		events = append(events, ev)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(events)
}
