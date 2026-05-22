package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

const maxRefers = 5

type ExcessiveRefersRule struct {
	Rule
}

func (r *ExcessiveRefersRule) Meta() Rule {
	return r.Rule
}

func (r *ExcessiveRefersRule) findReferencesNumber(nodes []*reader.RichNode) int{
    for i, child := range nodes {

        if child.Type == reader.NodeKeyword && strings.Contains(child.Value, "refer") {
			if i+1 < len(nodes) {

                nextNode := nodes[i+1]

                if nextNode.Type == reader.NodeVector {
                    return len(nextNode.Children)
                }
            }
        }
        if len(child.Children) > 0 {

            found := r.findReferencesNumber(child.Children)

            if found != 0 {
                return found
            }
        }
    }
    return 0
}

func (r *ExcessiveRefersRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *Finding {
	if len(node.Children) <= 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}
	
	if node.Children[0].Value == "ns" && r.findReferencesNumber(node.Children[1:]) > maxRefers {
		return &Finding{
            RuleID:   r.ID,
            Message:  fmt.Sprintf("The excessive number of explicit references in the %s namespace increases the risk of conflicts with other libraries or with future code.", node.Children[1].Value),
            Filepath: filepath,
            Location: node.Location,
            Severity: r.Severity,
        }
	}
    return nil
}

func init() {
	defaultRule := &ExcessiveRefersRule{
		Rule: Rule{
			ID:          "excessive-refers",
			Name:        "Excessive Refers",
			Description: "Excessive use of explicit references or `refer:all` pollutes the namespace, increasing the risk of collisions.",
			Severity:    SeverityWarning,
		},
	}
	RegisterRule(defaultRule)
}
