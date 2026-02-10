package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethanrcohen/ddcli/internal/api"
	"github.com/ethanrcohen/ddcli/internal/config"
	"github.com/ethanrcohen/ddcli/internal/output"
	"github.com/ethanrcohen/ddcli/internal/timeutil"
	"github.com/spf13/cobra"
)

var tracesGetCmd = &cobra.Command{
	Use:   "get <trace_id>",
	Short: "Get all spans for a trace",
	Long: `Fetch all spans belonging to a specific trace ID.

Examples:
  ddcli traces get abc123def456
  ddcli traces get abc123def456 --from 1h
  ddcli traces get abc123def456 --from 1h --output table`,
	Args: cobra.ExactArgs(1),
	RunE: runTracesGet,
}

func init() {
	tracesCmd.AddCommand(tracesGetCmd)
	tracesGetCmd.Flags().String("from", "15m", "Start time (relative: 15m, 1h, 24h, 7d or ISO 8601 timestamp)")
	tracesGetCmd.Flags().String("to", "now", "End time (relative or ISO 8601 timestamp)")
	tracesGetCmd.Flags().StringP("output", "o", "json", "Output format: json, table, or raw")
}

type tracesGetDeps struct {
	spansAPI api.SpansAPI
	now      time.Time
}

var tracesGetOverrides *tracesGetDeps

func runTracesGet(cmd *cobra.Command, args []string) error {
	traceID := args[0]
	query := fmt.Sprintf("trace_id:%s", traceID)

	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")
	outputFmt, _ := cmd.Flags().GetString("output")

	format, err := output.ParseFormat(outputFmt)
	if err != nil {
		return err
	}

	now := time.Now()
	var client api.SpansAPI

	if tracesGetOverrides != nil {
		now = tracesGetOverrides.now
		client = tracesGetOverrides.spansAPI
	} else {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
		client = api.NewClient(cfg.BaseURL(), cfg.APIKey, cfg.AppKey)
	}

	from, err := timeutil.ParseRelativeOrAbsolute(fromStr, now)
	if err != nil {
		return fmt.Errorf("parsing --from: %w", err)
	}
	to, err := timeutil.ParseRelativeOrAbsolute(toStr, now)
	if err != nil {
		return fmt.Errorf("parsing --to: %w", err)
	}

	params := api.SearchSpansParams{
		Query: query,
		From:  from,
		To:    to,
		Sort:  "timestamp",
		Limit: 1000,
	}

	// Exhaust all pages — a trace is only useful when complete
	var allSpans []api.SpanEntry
	for {
		resp, err := client.SearchSpans(context.Background(), params)
		if err != nil {
			return err
		}

		allSpans = append(allSpans, resp.Data...)

		if resp.Meta.Page.After == "" || len(resp.Data) == 0 {
			break
		}
		params.Cursor = resp.Meta.Page.After
	}

	result := &api.SpansListResponse{
		Data: allSpans,
	}

	w := cmd.OutOrStdout()
	formatter := output.NewSpansFormatter(format)
	return formatter.FormatSpans(w, result)
}

// SetTracesGetDeps sets test dependencies. Only for use in tests.
func SetTracesGetDeps(spansAPI api.SpansAPI, now time.Time) func() {
	tracesGetOverrides = &tracesGetDeps{
		spansAPI: spansAPI,
		now:      now,
	}
	return func() {
		tracesGetOverrides = nil
	}
}
