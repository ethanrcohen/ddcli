package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const installScript = "https://raw.githubusercontent.com/ethanrcohen/ddcli/main/install.sh"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ddcli to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current version: %s\n", buildVersion)
		fmt.Println("Downloading latest version...")

		sh := exec.Command("sh", "-c", fmt.Sprintf("curl -sSL %s | sh", installScript))
		sh.Stdout = os.Stdout
		sh.Stderr = os.Stderr
		if err := sh.Run(); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
