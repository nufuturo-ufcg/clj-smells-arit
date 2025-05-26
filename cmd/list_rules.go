package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/rules"
)

// listRulesCmd define o comando para listar todas as regras disponíveis
var listRulesCmd = &cobra.Command{
	Use:   "list-rules",
	Short: "List all available analysis rules",
	Long: `List all available analysis rules with their descriptions.

This command displays all registered rules that can be used for code analysis,
including their IDs, names, descriptions, and default severity levels.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Obtém todas as regras registradas
		allRules := rules.AllRules()
		
		if len(allRules) == 0 {
			fmt.Println("No rules are currently registered.")
			return nil
		}

		// Ordena as regras por ID para saída consistente
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
	// Adiciona o comando list-rules ao comando raiz
	rootCmd.AddCommand(listRulesCmd)
} 