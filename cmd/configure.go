package cmd

import (
	"fmt"

	"github.com/ethanrcohen/ddcli/internal/config"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Set Datadog API credentials",
	Long: `Configure your Datadog API key, application key, and site.

Examples:
  ddcli configure --api-key <key> --app-key <key>
  ddcli configure --api-key <key> --app-key <key> --site datadoghq.eu`,
	RunE: runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.Flags().String("api-key", "", "Datadog API key")
	configureCmd.Flags().String("app-key", "", "Datadog application key")
	configureCmd.Flags().String("site", "datadoghq.com", "Datadog site (e.g. datadoghq.com, datadoghq.eu)")
}

func runConfigure(cmd *cobra.Command, args []string) error {
	apiKey, _ := cmd.Flags().GetString("api-key")
	appKey, _ := cmd.Flags().GetString("app-key")
	site, _ := cmd.Flags().GetString("site")

	if apiKey == "" || appKey == "" {
		return fmt.Errorf("both --api-key and --app-key are required")
	}

	cfg := config.Config{
		APIKey: apiKey,
		AppKey: appKey,
		Site:   site,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Configuration saved successfully.")
	return nil
}
