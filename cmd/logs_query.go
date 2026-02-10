package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// addLogFilterFlags adds the common --service, --env, --host, --status flags
// to a cobra command. These are syntactic sugar that prepend to the query string.
func addLogFilterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("service", "s", "", "Filter by service name")
	cmd.Flags().StringP("env", "e", "", "Filter by environment (e.g. prod, staging)")
	cmd.Flags().String("host", "", "Filter by host")
	cmd.Flags().String("status", "", "Filter by log status (error, warn, info, debug)")
}

// buildQuery combines the positional query arg with the convenience filter flags
// into a single Datadog search query string.
func buildQuery(cmd *cobra.Command, args []string) string {
	var parts []string

	if service, _ := cmd.Flags().GetString("service"); service != "" {
		parts = append(parts, "service:"+service)
	}
	if env, _ := cmd.Flags().GetString("env"); env != "" {
		parts = append(parts, "env:"+env)
	}
	if host, _ := cmd.Flags().GetString("host"); host != "" {
		parts = append(parts, "host:"+host)
	}
	if status, _ := cmd.Flags().GetString("status"); status != "" {
		parts = append(parts, "status:"+status)
	}

	if len(args) > 0 && args[0] != "" {
		parts = append(parts, args[0])
	}

	if len(parts) == 0 {
		return "*"
	}
	return strings.Join(parts, " ")
}
