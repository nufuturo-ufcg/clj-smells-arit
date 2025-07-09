package cmd

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

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

			if ruleConfig := extractRuleConfig(rule); ruleConfig != nil {
				defaultConfig.RuleConfig[meta.ID] = ruleConfig
			}

			switch meta.ID {
			case "lazy-side-effects":
				defaultConfig.RuleConfig[meta.ID] = map[string]interface{}{
					"lazy_context_funcs": rules.DefaultLazyContextFunctions,
					"side_effect_funcs":  rules.DefaultSideEffectFunctions,
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

func extractRuleConfig(rule rules.RegisteredRule) map[string]interface{} {
	ruleValue := reflect.ValueOf(rule)
	if ruleValue.Kind() == reflect.Ptr {
		ruleValue = ruleValue.Elem()
	}

	if ruleValue.Kind() != reflect.Struct {
		return nil
	}

	ruleType := ruleValue.Type()
	ruleConfig := make(map[string]interface{})

	if rule.Meta().ID == "duplicated-code" {

		ruleConfig["enable-exact"] = true
		ruleConfig["enable-similar"] = true
		ruleConfig["exact-min-lines"] = 6
		ruleConfig["exact-min-tokens"] = 25
		ruleConfig["similar-min-lines"] = 3
		ruleConfig["similar-min-tokens"] = 15
		ruleConfig["max-cache-size"] = 10000
		ruleConfig["max-blocks-per-file"] = 1000
	} else {

		for i := 0; i < ruleValue.NumField(); i++ {
			field := ruleValue.Field(i)
			fieldType := ruleType.Field(i)

			if fieldType.Name == "Rule" {
				continue
			}

			yamlTag := fieldType.Tag.Get("yaml")
			jsonTag := fieldType.Tag.Get("json")

			var configKey string
			if yamlTag != "" && yamlTag != "-" {
				configKey = strings.Split(yamlTag, ",")[0]
			} else if jsonTag != "" && jsonTag != "-" {
				configKey = strings.Split(jsonTag, ",")[0]
			}

			if configKey == "" {
				continue
			}

			if field.CanInterface() {
				value := field.Interface()
				if !isZeroValue(field) {
					ruleConfig[configKey] = value
				}
			}
		}
	}

	if len(ruleConfig) == 0 {
		return nil
	}

	return ruleConfig
}

func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:

		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}

func init() {
	rootCmd.AddCommand(genconfCmd)
}
