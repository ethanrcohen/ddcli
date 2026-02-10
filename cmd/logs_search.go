package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ethanrcohen/ddcli/internal/api"
	"github.com/ethanrcohen/ddcli/internal/config"
	"github.com/ethanrcohen/ddcli/internal/output"
	"github.com/ethanrcohen/ddcli/internal/timeutil"
	"github.com/spf13/cobra"
)

var logsSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search logs with Datadog query syntax",
	Long: `Search Datadog logs using the standard query syntax.

The --service, --env, --host, and --status flags are convenience shortcuts
that get combined with the query argument. You can mix and match:

Examples:
  ddcli logs search --service payment --status error --from 1h
  ddcli logs search --service web-store --env prod "@duration:>5s" --from 15m
  ddcli logs search "service:payment status:error" --from 1h
  ddcli logs search --from "2024-01-01T00:00:00Z" --to "2024-01-02T00:00:00Z" -s payment
  ddcli logs search --status error --from 1h --output table
  ddcli logs search "*" --from 1h --output raw`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogsSearch,
}

func init() {
	logsCmd.AddCommand(logsSearchCmd)
	addLogFilterFlags(logsSearchCmd)
	logsSearchCmd.Flags().String("from", "15m", "Start time (relative: 15m, 1h, 24h, 7d or ISO 8601 timestamp)")
	logsSearchCmd.Flags().String("to", "now", "End time (relative or ISO 8601 timestamp)")
	logsSearchCmd.Flags().Int("limit", 50, "Maximum number of logs to return (max 1000)")
	logsSearchCmd.Flags().String("sort", "-timestamp", "Sort order: -timestamp (newest first) or timestamp (oldest first)")
	logsSearchCmd.Flags().StringSlice("indexes", nil, "Log indexes to search (default: all)")
	logsSearchCmd.Flags().StringP("output", "o", "json", "Output format: json, table, or raw")
}

// logsSearchDeps allows injecting dependencies for testing.
type logsSearchDeps struct {
	logsAPI api.LogsAPI
	now     time.Time
	output  *os.File
}

var logsSearchOverrides *logsSearchDeps

func runLogsSearch(cmd *cobra.Command, args []string) error {
	query := buildQuery(cmd, args)

	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")
	limit, _ := cmd.Flags().GetInt("limit")
	sort, _ := cmd.Flags().GetString("sort")
	indexes, _ := cmd.Flags().GetStringSlice("indexes")
	outputFmt, _ := cmd.Flags().GetString("output")

	format, err := output.ParseFormat(outputFmt)
	if err != nil {
		return err
	}

	now := time.Now()
	var client api.LogsAPI

	if logsSearchOverrides != nil {
		now = logsSearchOverrides.now
		client = logsSearchOverrides.logsAPI
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

	params := api.SearchLogsParams{
		Query:   query,
		From:    from,
		To:      to,
		Sort:    sort,
		Limit:   limit,
		Indexes: indexes,
	}

	// Collect all results, handling pagination
	var allEntries []api.LogEntry
	remaining := limit
	for {
		fetchLimit := remaining
		if fetchLimit > 1000 {
			fetchLimit = 1000
		}
		params.Limit = fetchLimit

		resp, err := client.SearchLogs(context.Background(), params)
		if err != nil {
			return err
		}

		allEntries = append(allEntries, resp.Data...)
		remaining -= len(resp.Data)

		if resp.Meta.Page.After == "" || remaining <= 0 || len(resp.Data) == 0 {
			break
		}
		params.Cursor = resp.Meta.Page.After
	}

	result := &api.LogsListResponse{
		Data: allEntries,
	}

	w := cmd.OutOrStdout()
	formatter := output.NewLogsFormatter(format)
	return formatter.FormatLogs(w, result)
}

// SetLogsSearchDeps sets test dependencies. Only for use in tests.
func SetLogsSearchDeps(logsAPI api.LogsAPI, now time.Time) func() {
	logsSearchOverrides = &logsSearchDeps{
		logsAPI: logsAPI,
		now:     now,
	}
	return func() {
		logsSearchOverrides = nil
	}
}

