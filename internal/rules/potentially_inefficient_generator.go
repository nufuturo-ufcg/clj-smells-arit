package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type PotentiallyInefficientGeneratorRule struct{}

func (r *PotentiallyInefficientGeneratorRule) Meta() Rule {
	return Rule{
		ID:          "inefficient-filter: inefficient-generator",
		Name:        "Potentially Inefficient Generator",
		Description: "Detects the use of generator functions like `gen/such-that` which might be inefficient if the predicate is rarely satisfied by the base generator, potentially leading to excessive generation attempts.",
		Severity:    SeverityHint,
	}
}

func (r *PotentiallyInefficientGeneratorRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	funcNode := node.Children[0]

	if funcNode.Type != reader.NodeSymbol || funcNode.Value != "gen/such-that" {
		return nil
	}

	meta := r.Meta()
	return &Finding{
		RuleID:   meta.ID,
		Message:  fmt.Sprintf("Using `gen/such-that` can be inefficient if the predicate is rarely satisfied by the base generator. Review if this usage might lead to excessive generation attempts."),
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
}

func init() {
	RegisterRule(&PotentiallyInefficientGeneratorRule{})
}
