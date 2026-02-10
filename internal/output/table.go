package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/ethanrcohen/ddcli/internal/api"
)

// TableFormatter outputs results as a human-readable table.
type TableFormatter struct{}

func (f *TableFormatter) FormatLogs(w io.Writer, resp *api.LogsListResponse) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TIMESTAMP\tSTATUS\tSERVICE\tHOST\tMESSAGE")
	fmt.Fprintln(tw, "---------\t------\t-------\t----\t-------")

	for _, entry := range resp.Data {
		msg := truncate(entry.Attributes.Message, 100)
		msg = strings.ReplaceAll(msg, "\n", " ")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			entry.Attributes.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Attributes.Status,
			entry.Attributes.Service,
			entry.Attributes.Host,
			msg,
		)
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(w, "\n(%d results)\n", len(resp.Data))
	return nil
}

func (f *TableFormatter) FormatAggregate(w io.Writer, resp *api.LogsAggregateResponse) error {
	if len(resp.Data.Buckets) == 0 {
		fmt.Fprintln(w, "(no results)")
		return nil
	}

	// Collect all group-by keys and compute keys from first bucket
	var groupKeys []string
	var computeKeys []string
	if len(resp.Data.Buckets) > 0 {
		for k := range resp.Data.Buckets[0].By {
			groupKeys = append(groupKeys, k)
		}
		for k := range resp.Data.Buckets[0].Computes {
			computeKeys = append(computeKeys, k)
		}
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Header
	headers := append(groupKeys, computeKeys...)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	dashes := make([]string, len(headers))
	for i, h := range headers {
		dashes[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(tw, strings.Join(dashes, "\t"))

	// Rows
	for _, bucket := range resp.Data.Buckets {
		var vals []string
		for _, k := range groupKeys {
			vals = append(vals, bucket.By[k])
		}
		for _, k := range computeKeys {
			vals = append(vals, fmt.Sprintf("%v", bucket.Computes[k]))
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	return tw.Flush()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
