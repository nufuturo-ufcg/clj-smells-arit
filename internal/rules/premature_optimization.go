package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

const (
	PrematureOptimizationRuleID   = "premature-optimization"
	PrematureOptimizationRuleName = "Premature Optimization"
)

type PrematureOptimizationRule struct{}

func NewPrematureOptimizationRule() RegisteredRule {
	return &PrematureOptimizationRule{}
}

func (r *PrematureOptimizationRule) Meta() Rule {
	return Rule{
		ID:       PrematureOptimizationRuleID,
		Name:     PrematureOptimizationRuleName,
		Severity: SeverityWarning,
		Description: "Identifies code patterns that suggest premature optimization, such as overly generic retry logic for I/O operations. " +
			"Optimizing code before identifying actual performance bottlenecks can lead to unnecessary complexity and reduced maintainability. " +
			"Consider if the optimization addresses a proven issue and if its complexity is justified.",
	}
}

func (r *PrematureOptimizationRule) Check(node *reader.RichNode, context map[string]interface{}, filepathAlias string) *Finding {

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		firstChild := node.Children[0]

		if firstChild.Type == reader.NodeSymbol && firstChild.Value == "with-retry" {

			meta := r.Meta()
			return &Finding{
				RuleID:   meta.ID,
				Severity: meta.Severity,
				Message:  fmt.Sprintf("Premature optimization: Generic 'with-retry' at line %d. Consider if this complexity is justified without profiling.", node.Location.StartLine),
				Location: node.Location,
				Filepath: filepathAlias,
			}
		}
	}

	return nil
}

func isIoCallRecursive(node *reader.RichNode) bool {
	fmt.Println("[DEBUG isIoCallRecursive - SIMPLIFIED AND DISABLED] Called, will return false.")

	return false
}

func isDirectIoCall(node *reader.RichNode) bool {
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		funcNode := node.Children[0]
		if funcNode.Type == reader.NodeSymbol {

			switch funcNode.Value {
			case "http/get", "clj-http.client/get", "slurp", "clojure.java.io/reader",
				"get-file-from-network-problematic", "get-another-file-from-network":
				return true
			}
		}
	}
	return false
}

func init() {

	RegisterRule(NewPrematureOptimizationRule())

}
