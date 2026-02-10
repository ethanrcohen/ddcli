package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethanrcohen/ddcli/internal/api"
	"github.com/ethanrcohen/ddcli/internal/config"
	"github.com/ethanrcohen/ddcli/internal/output"
	"github.com/ethanrcohen/ddcli/internal/timeutil"
	"github.com/spf13/cobra"
)

var logsAggregateCmd = &cobra.Command{
	Use:   "aggregate [query]",
	Short: "Aggregate logs and compute metrics",
	Long: `Aggregate log events and compute metrics like counts, averages, etc.

The --service, --env, --host, and --status flags are convenience shortcuts
that get combined with the query argument.

Examples:
  ddcli logs aggregate --status error --compute count --group-by service --from 24h
  ddcli logs aggregate --service payment --compute "avg:@duration" --from 1h
  ddcli logs aggregate "status:error" --compute count --group-by service --from 24h
  ddcli logs aggregate "*" --compute "avg:@duration" --group-by service --from 1h`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogsAggregate,
}

func init() {
	logsCmd.AddCommand(logsAggregateCmd)
	addLogFilterFlags(logsAggregateCmd)
	logsAggregateCmd.Flags().String("from", "1h", "Start time (relative or ISO 8601)")
	logsAggregateCmd.Flags().String("to", "now", "End time (relative or ISO 8601)")
	logsAggregateCmd.Flags().String("compute", "count", "Aggregation: count, avg:<metric>, sum:<metric>, min:<metric>, max:<metric>")
	logsAggregateCmd.Flags().String("group-by", "", "Facet to group by (e.g. service, status, host)")
	logsAggregateCmd.Flags().StringP("output", "o", "json", "Output format: json, table, or raw")
}

var logsAggregateOverrides *logsSearchDeps

func runLogsAggregate(cmd *cobra.Command, args []string) error {
	query := buildQuery(cmd, args)

	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")
	computeStr, _ := cmd.Flags().GetString("compute")
	groupByStr, _ := cmd.Flags().GetString("group-by")
	outputFmt, _ := cmd.Flags().GetString("output")

	format, err := output.ParseFormat(outputFmt)
	if err != nil {
		return err
	}

	now := time.Now()
	var client api.LogsAPI

	if logsAggregateOverrides != nil {
		now = logsAggregateOverrides.now
		client = logsAggregateOverrides.logsAPI
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

	compute, err := parseCompute(computeStr)
	if err != nil {
		return err
	}

	params := api.AggregateLogsParams{
		Query:   query,
		From:    from,
		To:      to,
		Compute: []api.AggregateCompute{compute},
	}

	if groupByStr != "" {
		params.GroupBy = []api.AggregateGroupBy{
			{
				Facet: groupByStr,
				Limit: 10,
				Sort: &api.AggregateGroupSort{
					Aggregation: compute.Aggregation,
					Order:       "desc",
				},
			},
		}
	}

	resp, err := client.AggregateLogs(context.Background(), params)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	formatter := output.NewAggregateFormatter(format)
	return formatter.FormatAggregate(w, resp)
}

// parseCompute parses a compute string like "count", "avg:@duration", "sum:@bytes".
func parseCompute(s string) (api.AggregateCompute, error) {
	parts := strings.SplitN(s, ":", 2)
	agg := parts[0]

	switch agg {
	case "count":
		return api.AggregateCompute{Aggregation: "count", Type: "total"}, nil
	case "avg", "sum", "min", "max", "pct":
		if len(parts) < 2 || parts[1] == "" {
			return api.AggregateCompute{}, fmt.Errorf("aggregation %q requires a metric (e.g. %s:@duration)", agg, agg)
		}
		return api.AggregateCompute{Aggregation: agg, Metric: parts[1], Type: "total"}, nil
	default:
		return api.AggregateCompute{}, fmt.Errorf("unknown aggregation %q (use count, avg, sum, min, max, pct)", agg)
	}
}

// SetLogsAggregateDeps sets test dependencies for the aggregate command.
func SetLogsAggregateDeps(logsAPI api.LogsAPI, now time.Time) func() {
	logsAggregateOverrides = &logsSearchDeps{
		logsAPI: logsAPI,
		now:     now,
	}
	return func() {
		logsAggregateOverrides = nil
	}
}
