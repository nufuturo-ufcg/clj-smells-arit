package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type NestedAtomsRule struct {
	Rule
}

func (r *NestedAtomsRule) Meta() Rule {
	return r.Rule
}

func (r *NestedAtomsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type == reader.NodeList && len(node.Children) > 1 {
		if node.Children[0].Type == reader.NodeSymbol {
			if r.isAtom(node) {
				if r.hasAtoms(node) {
					return &Finding{
						RuleID: r.ID,
						Message: fmt.Sprintf("Found nested Atom/Ref/Volatile inside an Atom. " +
"Breaking atomic state management - inner updates won't trigger watchers."),
						Filepath: filepath,
						Location: node.Location,
						Severity: r.Severity,
					}
				}
			}
		}
	}
	return nil
}

func (r *NestedAtomsRule) hasAtoms(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	atoms := r.filterNodes(node.Children, r.isAtomOrHasAtoms)
	return len(atoms) > 0
}

func (r *NestedAtomsRule) isAtomOrHasAtoms(node *reader.RichNode) bool {
    return r.isAtom(node) || r.hasAtoms(node)
}

func (r *NestedAtomsRule) isAtom(node *reader.RichNode) bool {
	return node.Type == reader.NodeList &&
		len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "atom" || node.Children[0].Value == "volatile!" || node.Children[0].Value == "ref")
}

func (r *NestedAtomsRule) filterNodes(nodes []*reader.RichNode, predicate func(*reader.RichNode) bool) []*reader.RichNode {
	result := []*reader.RichNode{}
	for _, node := range nodes {
		if predicate(node) {
			result = append(result, node)
		}
	}

	return result
}

func init() {
	defaultRule := &NestedAtomsRule{
		Rule: Rule{
			ID:          "nested-atoms",
			Name:        "Nested Atoms",
			Description: "Detects an Atom or other managed reference (like a Volatile or Ref) inside another Atom",
			Severity:    SeverityWarning,
		},
	}
	RegisterRule(defaultRule)
}
