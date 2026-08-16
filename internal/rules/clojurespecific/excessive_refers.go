package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ExcessiveRefersRule struct {
	rules.Rule
	MaxExplicitRefers int `json:"max_explicit_refers" yaml:"max_explicit_refers"`
}

func (r *ExcessiveRefersRule) Meta() rules.Rule {
	return r.Rule
}

func (r *ExcessiveRefersRule) countExplicitReferences(nodes []*reader.RichNode) int {
	total := 0

	for i, child := range nodes {
		isExplicitImport := child.Type == reader.NodeKeyword &&
			(child.Value == ":refer" || child.Value == ":only")
		if isExplicitImport && i+1 < len(nodes) {
			nextNode := nodes[i+1]
			if nextNode.Type == reader.NodeVector {
				total += len(nextNode.Children)
			}
		}
		if len(child.Children) > 0 {
			total += r.countExplicitReferences(child.Children)
		}
	}
	return total
}

func (r *ExcessiveRefersRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {
	if len(node.Children) <= 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	if node.Children[0].Value == "ns" {
		totalExplicitRefers := r.countExplicitReferences(node.Children[1:])

		if totalExplicitRefers >= r.MaxExplicitRefers {
			return &rules.Finding{
				RuleID: r.ID,
				Message: fmt.Sprintf(
					"Namespace `%s` explicitly refers %d Vars, meeting the configured review threshold of %d. The default threshold (24) was calibrated as the mean plus two standard deviations across 430 repositories. This is a statistical outlier signal, not proof of a conflict; the developer should evaluate whether the import surface is appropriate.",
					node.Children[1].Value, totalExplicitRefers, r.MaxExplicitRefers,
				),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}
	return nil
}

func init() {
	defaultRule := &ExcessiveRefersRule{
		Rule: rules.Rule{
			ID:          "excessive-refers",
			Name:        "Excessive Refers",
			Description: "Detects statistical outliers in the total number of Vars explicitly imported through :refer [...] or :use ... :only [...]. The default inclusive threshold of 24 was calibrated as the mean plus two standard deviations across 430 repositories. Unrestricted imports such as :refer :all belong to implicit-namespace-dependencies.",
			Severity:    rules.SeverityWarning,
		},
		MaxExplicitRefers: 24,
	}
	rules.RegisterRule(defaultRule)
}
