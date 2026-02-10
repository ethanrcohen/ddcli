package cmd

import (
	"fmt"
	"os"

	"github.com/ethanrcohen/ddcli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the ddcli version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ddcli %s (commit %s, built %s)\n", buildVersion, commit, date)

		if buildVersion != "dev" {
			result := version.CheckForUpdate(buildVersion)
			if notice := version.FormatNotice(result); notice != "" {
				fmt.Fprintln(os.Stderr, notice)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
