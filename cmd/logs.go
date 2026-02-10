package cmd

import "github.com/spf13/cobra"

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Search, aggregate, and tail Datadog logs",
	Long: `Commands for interacting with Datadog Log Management.

Uses Datadog's log search query syntax. Common patterns:
  service:my-service          Filter by service
  status:error                Filter by status (error, warn, info, debug)
  host:my-host                Filter by host
  @duration:>5s               Filter by custom attribute
  "exact phrase"              Match exact phrase
  service:web AND status:error   Boolean operators (AND, OR, NOT)`,
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
