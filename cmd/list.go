package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/rules"
)

var viewRulesCmd = &cobra.Command{
	Use:   "list-rules",
	Short: "List a simplified list of all available analysis rules",
	Long:  `List a simplified list of all available analysis rules.`,
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
			fmt.Printf("Name: %s\n", meta.Name)
			fmt.Println("---")
		}

		return nil
	},
}

func init() {

	rootCmd.AddCommand(viewRulesCmd)
}
