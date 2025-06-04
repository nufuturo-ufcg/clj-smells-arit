package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type ProductionDoallRule struct {
	Rule
	AllowInTests   bool `json:"allow_in_tests" yaml:"allow_in_tests"`
	AllowInDevCode bool `json:"allow_in_dev_code" yaml:"allow_in_dev_code"`
	AllowInREPL    bool `json:"allow_in_repl" yaml:"allow_in_repl"`
}

func (r *ProductionDoallRule) Meta() Rule {
	return r.Rule
}

func (r *ProductionDoallRule) isTestFile(filepath string) bool {

	testIndicators := []string{
		"test",
		"spec",
		"check",
		"_test",
		"-test",
		"test_",
		"test-",
	}

	for _, indicator := range testIndicators {
		if contains(filepath, indicator) {
			return true
		}
	}

	return false
}

func (r *ProductionDoallRule) isDevCode(filepath string) bool {
	devIndicators := []string{
		"dev",
		"development",
		"debug",
		"repl",
		"scratch",
		"playground",
		"example",
		"demo",
	}

	for _, indicator := range devIndicators {
		if contains(filepath, indicator) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				anySubstring(s, substr)))
}

func anySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (r *ProductionDoallRule) isInREPLContext(node *reader.RichNode, filepath string) bool {

	replIndicators := []string{
		"repl",
		"user",
		"scratch",
	}

	for _, indicator := range replIndicators {
		if contains(filepath, indicator) {
			return true
		}
	}

	return false
}

func (r *ProductionDoallRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 1 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || firstChild.Value != "doall" {
		return nil
	}

	if r.AllowInTests && r.isTestFile(filepath) {
		return nil
	}

	if r.AllowInDevCode && r.isDevCode(filepath) {
		return nil
	}

	if r.AllowInREPL && r.isInREPLContext(node, filepath) {
		return nil
	}

	contextDescription := r.getContextDescription(node, context)

	message := fmt.Sprintf(
		"Usage of 'doall' detected in production code%s. "+
			"'doall' forces realization of lazy sequences which can cause memory issues and performance problems. "+
			"Consider using eager operations (mapv, into, vec) or restructuring to avoid forcing evaluation. "+
			"If this is intentional for debugging/testing, consider moving to test files or dev-specific code.",
		contextDescription,
	)

	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func (r *ProductionDoallRule) getContextDescription(node *reader.RichNode, context map[string]interface{}) string {

	if parent, ok := context["parent"]; ok {
		if parentNode, ok := parent.(*reader.RichNode); ok {
			if parentNode.Type == reader.NodeList && len(parentNode.Children) > 0 {
				if parentFirstChild := parentNode.Children[0]; parentFirstChild.Type == reader.NodeSymbol {
					switch parentFirstChild.Value {
					case "defn", "defn-":
						if len(parentNode.Children) > 1 && parentNode.Children[1].Type == reader.NodeSymbol {
							return fmt.Sprintf(" in function '%s'", parentNode.Children[1].Value)
						}
						return " in function definition"
					case "let", "when", "if":
						return fmt.Sprintf(" in %s form", parentFirstChild.Value)
					case "map", "mapcat", "filter":
						return fmt.Sprintf(" in %s operation (nested lazy operation)", parentFirstChild.Value)
					}
				}
			}
		}
	}

	if len(node.Children) > 1 {
		argNode := node.Children[1]
		if argNode.Type == reader.NodeList && len(argNode.Children) > 0 {
			if firstArg := argNode.Children[0]; firstArg.Type == reader.NodeSymbol {
				switch firstArg.Value {
				case "map", "filter", "remove", "mapcat":
					return fmt.Sprintf(" on %s result", firstArg.Value)
				case "for":
					return " on list comprehension result"
				}
			}
		}
	}

	return ""
}

func init() {
	defaultRule := &ProductionDoallRule{
		Rule: Rule{
			ID:          "production-doall",
			Name:        "Production doall Usage",
			Description: "Detects usage of 'doall' in production code. 'doall' forces realization of lazy sequences which can cause memory issues and performance problems. Consider using eager operations or restructuring code to avoid forcing evaluation.",
			Severity:    SeverityWarning,
		},
		AllowInTests:   false,
		AllowInDevCode: false,
		AllowInREPL:    true,
	}

	RegisterRule(defaultRule)
}
