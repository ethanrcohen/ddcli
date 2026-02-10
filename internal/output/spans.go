package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ethanrcohen/ddcli/internal/api"
)

func (f *JSONFormatter) FormatSpans(w io.Writer, resp *api.SpansListResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func (f *RawFormatter) FormatSpans(w io.Writer, resp *api.SpansListResponse) error {
	for _, entry := range resp.Data {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
	}
	return nil
}

func (f *TableFormatter) FormatSpans(w io.Writer, resp *api.SpansListResponse) error {
	if len(resp.Data) == 0 {
		fmt.Fprintln(w, "(no spans found)")
		return nil
	}

	// Build the span tree
	tree := buildSpanTree(resp.Data)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVICE\tRESOURCE\tTYPE\tDURATION\tSPAN_ID")
	fmt.Fprintln(tw, "-------\t--------\t----\t--------\t-------")

	// Walk the tree via DFS
	for _, root := range tree.roots {
		printSpanNode(tw, root, tree.children, 0)
	}

	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(w, "\n(%d spans)\n", len(resp.Data))
	return nil
}

type spanTree struct {
	roots    []*api.SpanEntry
	children map[string][]*api.SpanEntry
}

func buildSpanTree(spans []api.SpanEntry) spanTree {
	byID := make(map[string]*api.SpanEntry, len(spans))
	children := make(map[string][]*api.SpanEntry)

	for i := range spans {
		byID[spans[i].Attributes.SpanID] = &spans[i]
	}

	var roots []*api.SpanEntry
	for i := range spans {
		s := &spans[i]
		parentID := s.Attributes.ParentID
		if parentID == "" || byID[parentID] == nil {
			roots = append(roots, s)
		} else {
			children[parentID] = append(children[parentID], s)
		}
	}

	// Sort roots and children by timestamp for stable output
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Attributes.StartTimestamp.Before(roots[j].Attributes.StartTimestamp)
	})
	for k := range children {
		c := children[k]
		sort.Slice(c, func(i, j int) bool {
			return c[i].Attributes.StartTimestamp.Before(c[j].Attributes.StartTimestamp)
		})
	}

	return spanTree{roots: roots, children: children}
}

func printSpanNode(w io.Writer, span *api.SpanEntry, children map[string][]*api.SpanEntry, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n",
		indent,
		span.Attributes.Service,
		truncate(span.Attributes.ResourceName, 60),
		span.Attributes.OperationName,
		formatDuration(span.Attributes.Duration()),
		span.Attributes.SpanID,
	)

	for _, child := range children[span.Attributes.SpanID] {
		printSpanNode(w, child, children, depth+1)
	}
}

// formatDuration formats nanoseconds into a human-readable string.
func formatDuration(ns int64) string {
	if ns < 1000 {
		return fmt.Sprintf("%dns", ns)
	}
	us := float64(ns) / 1000
	if us < 1000 {
		return fmt.Sprintf("%.0fus", us)
	}
	ms := us / 1000
	if ms < 1000 {
		return fmt.Sprintf("%.1fms", ms)
	}
	s := ms / 1000
	if s < 60 {
		return fmt.Sprintf("%.2fs", s)
	}
	m := s / 60
	return fmt.Sprintf("%.1fm", m)
}
