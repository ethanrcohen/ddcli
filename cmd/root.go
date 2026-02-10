package cmd

import (
	"fmt"
	"os"

	"github.com/ethanrcohen/ddcli/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Set via ldflags at build time by goreleaser.
	buildVersion = "dev"
	commit       = "none"
	date         = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "ddcli",
	Short: "Datadog CLI for AI agents and humans",
	Long: `ddcli is a command-line interface for interacting with Datadog.
Designed for use by AI agents and humans alike, it provides
structured access to logs, metrics, and more.

Configure authentication:
  ddcli configure --api-key <key> --app-key <key>

Or set environment variables:
  export DD_API_KEY=<key>
  export DD_APP_KEY=<key>
  export DD_SITE=datadoghq.com  # optional, defaults to datadoghq.com`,
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if buildVersion == "dev" {
			return
		}
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return
		}
		result := version.CheckForUpdate(buildVersion)
		if notice := version.FormatNotice(result); notice != "" {
			fmt.Fprintln(os.Stderr, notice)
		}
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
