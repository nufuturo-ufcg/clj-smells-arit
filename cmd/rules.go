package cmd

import (
	"github.com/spf13/cobra"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage and query analysis rules",
	Long:  `Provides commands to list and get detailed information about available ARIT analysis rules.`,
}

func init() {
	rootCmd.AddCommand(rulesCmd)
}
