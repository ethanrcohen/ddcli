package output

import (
	"encoding/json"
	"io"

	"github.com/ethanrcohen/ddcli/internal/api"
)

// JSONFormatter outputs results as JSON.
type JSONFormatter struct{}

func (f *JSONFormatter) FormatLogs(w io.Writer, resp *api.LogsListResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func (f *JSONFormatter) FormatAggregate(w io.Writer, resp *api.LogsAggregateResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}
