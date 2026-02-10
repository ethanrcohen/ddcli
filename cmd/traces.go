package cmd

import "github.com/spf13/cobra"

var tracesCmd = &cobra.Command{
	Use:   "traces",
	Short: "Fetch and inspect Datadog traces",
	Long: `Commands for interacting with Datadog APM traces.

Retrieve all spans for a specific trace by its trace ID.`,
}

func init() {
	rootCmd.AddCommand(tracesCmd)
}
