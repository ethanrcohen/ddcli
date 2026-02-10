package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/ethan/ddcli/internal/api"
	"github.com/ethan/ddcli/internal/config"
	"github.com/ethan/ddcli/internal/output"
	"github.com/spf13/cobra"
)

var logsTailCmd = &cobra.Command{
	Use:   "tail [query]",
	Short: "Stream logs in real time",
	Long: `Continuously poll for new logs matching the given query.

The --service, --env, --host, and --status flags are convenience shortcuts
that get combined with the query argument.

Examples:
  ddcli logs tail --service payment
  ddcli logs tail --service payment --status error --output raw
  ddcli logs tail --host web-1 --interval 5s
  ddcli logs tail "service:payment status:error"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogsTail,
}

func init() {
	logsCmd.AddCommand(logsTailCmd)
	addLogFilterFlags(logsTailCmd)
	logsTailCmd.Flags().StringP("output", "o", "raw", "Output format: json, table, or raw")
	logsTailCmd.Flags().Duration("interval", 2*time.Second, "Poll interval")
}

func runLogsTail(cmd *cobra.Command, args []string) error {
	query := buildQuery(cmd, args)

	outputFmt, _ := cmd.Flags().GetString("output")
	interval, _ := cmd.Flags().GetDuration("interval")

	format, err := output.ParseFormat(outputFmt)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	client := api.NewClient(cfg.BaseURL(), cfg.APIKey, cfg.AppKey)
	formatter := output.NewLogsFormatter(format)
	w := cmd.OutOrStdout()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Start tailing from now
	lastSeen := time.Now()
	var lastID string

	fmt.Fprintf(os.Stderr, "Tailing logs matching %q (Ctrl+C to stop)...\n", query)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			params := api.SearchLogsParams{
				Query: query,
				From:  lastSeen,
				To:    time.Now(),
				Sort:  "timestamp",
				Limit: 100,
			}

			resp, err := client.SearchLogs(ctx, params)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error polling logs: %v\n", err)
				continue
			}

			// Filter out the last-seen log to avoid duplicates
			var newEntries []api.LogEntry
			for _, entry := range resp.Data {
				if entry.ID != lastID {
					newEntries = append(newEntries, entry)
				}
			}

			if len(newEntries) > 0 {
				display := &api.LogsListResponse{Data: newEntries}
				if err := formatter.FormatLogs(w, display); err != nil {
					fmt.Fprintf(os.Stderr, "Error formatting: %v\n", err)
				}
				last := newEntries[len(newEntries)-1]
				lastSeen = last.Attributes.Timestamp
				lastID = last.ID
			}
		}
	}
}
