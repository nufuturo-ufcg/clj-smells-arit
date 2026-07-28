package clojurespecific

import (
	"fmt"
	"github.com/thlaurentino/arit/internal/rules"

	"github.com/thlaurentino/arit/internal/reader"
)

var recordFUnctions []string

type NonIdiomaticRecordConstructionRule struct {
	rules.Rule
}

func (r *NonIdiomaticRecordConstructionRule) Meta() rules.Rule {
	return r.Rule
}

func (r *NonIdiomaticRecordConstructionRule) verifiesPositionalConstructor(value string) string {
	for _, function := range recordFUnctions {
		if value == "->"+function || value == function+"." {
			return function
		}
	}
	return ""
}

func (r *NonIdiomaticRecordConstructionRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {

	if node.Type != reader.NodeList || len(node.Children) <= 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	firstChild := node.Children[0].Value

	if firstChild == "defrecord" && node.Children[1].Type == reader.NodeSymbol {
		recordFUnctions = append(recordFUnctions, node.Children[1].Value)
	} else {

		function := r.verifiesPositionalConstructor(firstChild)

		if function != "" {
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Using a positional constructor to instantiate the defrecord instead of map->%s", function),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &NonIdiomaticRecordConstructionRule{
		Rule: rules.Rule{
			ID:          "non-idiomatic-record-construction",
			Name:        "Non-idiomatic Record Construction",
			Description: "Using Java's positional interpolate constructor to instantiate a defrecord causes the code to break silently if the fields are reordered.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}
