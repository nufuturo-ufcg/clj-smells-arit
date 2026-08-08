package clojurespecific

import (
	"fmt"
	"strings"

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

func (r *ExcessiveRefersRule) checkReferences(nodes []*reader.RichNode) (bool, int) {
	hasReferAll := false
	maxVectorLen := 0
	
	for i, child := range nodes {
		if child.Type == reader.NodeKeyword && strings.Contains(child.Value, "refer") {
			if i+1 < len(nodes) {
				nextNode := nodes[i+1]
				if nextNode.Type == reader.NodeKeyword && nextNode.Value == ":all" {
					hasReferAll = true
				} else if nextNode.Type == reader.NodeVector {
					if len(nextNode.Children) > maxVectorLen {
						maxVectorLen = len(nextNode.Children)
					}
				}
			}
		}
		if len(child.Children) > 0 {
			childHasReferAll, childMaxVectorLen := r.checkReferences(child.Children)
			if childHasReferAll {
				hasReferAll = true
			}
			if childMaxVectorLen > maxVectorLen {
				maxVectorLen = childMaxVectorLen
			}
		}
	}
	return hasReferAll, maxVectorLen
}

func (r *ExcessiveRefersRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {
	if len(node.Children) <= 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	if node.Children[0].Value == "ns" {
		hasReferAll, maxVectorLen := r.checkReferences(node.Children[1:])
		
		if hasReferAll {
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Usage of `:refer :all` found in the %s namespace. This pollutes the namespace and increases the risk of collisions.", node.Children[1].Value),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
		
		if maxVectorLen > r.MaxExplicitRefers {
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("The excessive number of explicit references (%d) in the %s namespace increases the risk of conflicts with other libraries or with future code.", maxVectorLen, node.Children[1].Value),
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
			Description: "Excessive use of explicit references or `refer:all` pollutes the namespace, increasing the risk of collisions.",
			Severity:    rules.SeverityWarning,
		},
		MaxExplicitRefers: 6,
	}
	rules.RegisterRule(defaultRule)
}
