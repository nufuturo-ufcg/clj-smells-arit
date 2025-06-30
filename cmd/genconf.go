package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/rules"
	"gopkg.in/yaml.v2"
)

var genconfCmd = &cobra.Command{
	Use:   "genconf",
	Short: "Generate a default .arit.yaml configuration file.",
	Long: `Generate a default .arit.yaml configuration file.

This command creates a .arit.yaml file in the current directory
with all available rules listed. You can then edit this file
to enable or disable rules and configure their parameters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		defaultConfig := config.Config{
			EnabledRules: make(map[string]bool),
			RuleConfig:   make(map[string]config.RuleSettings),
		}

		allRules := rules.AllRules()
		sort.Slice(allRules, func(i, j int) bool {
			return allRules[i].Meta().ID < allRules[j].Meta().ID
		})

		fmt.Println("Generating default configuration for all available rules...")

		for _, rule := range allRules {
			meta := rule.Meta()
			defaultConfig.EnabledRules[meta.ID] = true

			switch meta.ID {
			case "lazy-side-effects":
				defaultConfig.RuleConfig[meta.ID] = map[string]interface{}{
					"lazy_context_funcs": rules.DefaultLazyContextFunctions,
					"side_effect_funcs":  rules.DefaultSideEffectFunctions,
				}
			case "duplicated-code-global", "duplicated-code-fingerprint":
				defaultConfig.RuleConfig[meta.ID] = map[string]interface{}{
					"min_lines":  3,
					"min_tokens": 15,
				}
			case "long-function":
				defaultConfig.RuleConfig[meta.ID] = map[string]interface{}{
					"max_lines": 50,
				}
			case "long-parameter-list":
				defaultConfig.RuleConfig[meta.ID] = map[string]interface{}{
					"max_params": 5,
				}
			}
		}

		yamlData, err := yaml.Marshal(&defaultConfig)
		if err != nil {
			return fmt.Errorf("error marshalling default config to YAML: %w", err)
		}

		headerComment := []byte(
			`# Arit configuration file.
#
# You can enable or disable rules by setting them to true or false in the 'enabled_rules' section.
# Specific parameters for rules can be configured in the 'rule_config' section.
#
`)
		finalYamlData := append(headerComment, yamlData...)

		filePath := ".arit.yaml"
		if err := os.WriteFile(filePath, finalYamlData, 0644); err != nil {
			return fmt.Errorf("error writing to %s: %w", filePath, err)
		}

		fmt.Printf("Successfully created default configuration file at %s\n", filePath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(genconfCmd)
}
