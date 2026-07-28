package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type ProductionDoallRule struct {
	rules.Rule
	AllowInTests   bool `json:"allow_in_tests" yaml:"allow_in_tests"`
	AllowInDevCode bool `json:"allow_in_dev_code" yaml:"allow_in_dev_code"`
	AllowInREPL    bool `json:"allow_in_repl" yaml:"allow_in_repl"`
}

func (r *ProductionDoallRule) Meta() rules.Rule {
	return r.Rule
}

func (r *ProductionDoallRule) isTestFile(filepath string) bool {
	if strings.Contains(filepath, "internal/test/data/") {
		return false
	}
	return strings.HasSuffix(filepath, "_test.clj") || strings.HasSuffix(filepath, "-test.clj") || strings.Contains(filepath, "/int-test/")
}

func (r *ProductionDoallRule) isDevCode(filepath string) bool {
	if strings.Contains(filepath, "internal/test/data/") {
		return false
	}
	return strings.Contains(filepath, "/dev/") || strings.HasSuffix(filepath, "user.clj")
}

func (r *ProductionDoallRule) isInREPLContext(node *reader.RichNode, filepath string) bool {

	replIndicators := []string{
		"repl",
		"user",
		"scratch",
	}

	for _, indicator := range replIndicators {
		if strings.Contains(filepath, indicator) {
			return true
		}
	}

	return false
}

func (r *ProductionDoallRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {

	if strings.HasSuffix(filepath, "project.clj") || strings.HasSuffix(filepath, "deps.edn") {
		return nil
	}

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

	return &rules.Finding{
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
		Rule: rules.Rule{
			ID:          "production-doall",
			Name:        "Production doall Usage",
			Description: "Detects usage of 'doall' in production code. 'doall' forces realization of lazy sequences which can cause memory issues and performance problems. Consider using eager operations or restructuring code to avoid forcing evaluation.",
			Severity:    rules.SeverityWarning,
		},
		AllowInTests:   true,
		AllowInDevCode: true,
		AllowInREPL:    true,
	}

	rules.RegisterRule(defaultRule)
}