package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/rules"
)

var listRulesCmd = &cobra.Command{
	Use:   "info",
	Short: "Detailed information about analysis rules",
	Long: `Display all registered rules that can be used for code analysis,
including their IDs, names, descriptions, and default severity levels.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		allRules := rules.AllRules()

		if len(allRules) == 0 {
			fmt.Println("No rules are currently registered.")
			return nil
		}

		sort.Slice(allRules, func(i, j int) bool {
			return allRules[i].Meta().ID < allRules[j].Meta().ID
		})

		fmt.Printf("Available Analysis Rules (%d total):\n\n", len(allRules))

		for _, rule := range allRules {
			meta := rule.Meta()
			fmt.Printf("ID: %s\n", meta.ID)
			fmt.Printf("Name: %s\n", meta.Name)
			fmt.Printf("Severity: %s\n", meta.Severity)
			fmt.Printf("Description: %s\n", meta.Description)
			fmt.Println("---")
		}

		return nil
	},
}

func init() {
	rulesCmd.AddCommand(listRulesCmd)
}
