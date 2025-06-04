package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

const compositionFunctionArgsThreshold = 3

type OverabstractedCompositionRule struct{}

func (r *OverabstractedCompositionRule) Meta() Rule {

	return Rule{
		ID:          "overabstracted-composition",
		Name:        "Overabstracted Composition",
		Description: fmt.Sprintf("Detects compositions using `comp` that involve more than %d functions. Long chains of composition can sometimes hinder readability and debugging. Consider breaking down complex compositions.", compositionFunctionArgsThreshold),
		Severity:    SeverityHint,
	}
}

func (r *OverabstractedCompositionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol || firstElement.Value != "comp" {
		return nil
	}

	numberOfFunctions := len(node.Children) - 1

	if numberOfFunctions > compositionFunctionArgsThreshold {
		meta := r.Meta()
		message := fmt.Sprintf("The `comp` form at this location is composing %d functions, which exceeds the recommended maximum of %d. This can make the code harder to understand and debug. Consider refactoring into smaller, named compositions or using a threading macro if appropriate.", numberOfFunctions, compositionFunctionArgsThreshold)

		return &Finding{
			RuleID:   meta.ID,
			Message:  message,
			Filepath: filepath,
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

func init() {
	RegisterRule(&OverabstractedCompositionRule{})
}
