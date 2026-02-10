package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethanrcohen/ddcli/internal/api"
)

// RawFormatter outputs just the log message field, one per line.
type RawFormatter struct{}

func (f *RawFormatter) FormatLogs(w io.Writer, resp *api.LogsListResponse) error {
	for _, entry := range resp.Data {
		fmt.Fprintln(w, entry.Attributes.Message)
	}
	return nil
}

func (f *RawFormatter) FormatAggregate(w io.Writer, resp *api.LogsAggregateResponse) error {
	// For aggregate, raw format falls back to compact JSON (one bucket per line)
	for _, bucket := range resp.Data.Buckets {
		data, err := json.Marshal(bucket)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
	}
	return nil
}
