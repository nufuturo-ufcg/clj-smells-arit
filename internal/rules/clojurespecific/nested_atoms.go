package clojurespecific

import (

	"github.com/thlaurentino/arit/internal/rules"

	"github.com/thlaurentino/arit/internal/reader"
)

type NestedAtomsRule struct {
	rules.Rule
}

func (r *NestedAtomsRule) Meta() rules.Rule {
	return r.Rule
}

func (r *NestedAtomsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	head := node.Children[0]
	if head.Type != reader.NodeSymbol {
		return nil
	}

	if isStateCreation(node) {
		for _, child := range node.Children[1:] {
			if found := findStateCreation(child); found != nil {
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  "Found nested Atom/Ref/Volatile/Agent inside a stateful reference.",
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	if isStateUpdate(node) {
		for _, child := range node.Children[1:] {
			if found := findStateCreation(child); found != nil {
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  "Found stateful reference creation inside a state update function (swap!, reset!, etc).",
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	if head.Value == "let" && len(node.Children) >= 3 {
		bindings := node.Children[1]
		if bindings.Type == reader.NodeVector {
			for i := 0; i < len(bindings.Children)-1; i += 2 {
				sym := bindings.Children[i]
				val := bindings.Children[i+1]
				if sym.Type == reader.NodeSymbol {
					if created := findStateCreation(val); created != nil {
						for _, bodyNode := range node.Children[2:] {
							if updateNode := findStateUpdateUsingSymbol(bodyNode, sym.Value); updateNode != nil {
								return &rules.Finding{
									RuleID:   r.ID,
									Message:  "Found stateful reference created in let binding being inserted into another stateful reference.",
									Filepath: filepath,
									Location: node.Location,
									Severity: r.Severity,
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func isStateCreationSymbol(s string) bool {
	return s == "atom" || s == "volatile!" || s == "ref" || s == "agent"
}

func isStateUpdateSymbol(s string) bool {
	switch s {
	case "swap!", "reset!", "send", "send-off", "alter", "commute", "vreset!", "vswap!":
		return true
	}
	return false
}

func isStateCreation(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	return head.Type == reader.NodeSymbol && isStateCreationSymbol(head.Value)
}

func isStateUpdate(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	return head.Type == reader.NodeSymbol && isStateUpdateSymbol(head.Value)
}

func findStateCreation(node *reader.RichNode) *reader.RichNode {
	if node == nil {
		return nil
	}
	if isStateCreation(node) {
		return node
	}
	for _, child := range node.Children {
		if found := findStateCreation(child); found != nil {
			return found
		}
	}
	return nil
}

func findStateUpdateUsingSymbol(node *reader.RichNode, sym string) *reader.RichNode {
	if node == nil {
		return nil
	}
	if isStateUpdate(node) {
		if usesSymbol(node, sym) {
			return node
		}
	}
	for _, child := range node.Children {
		if found := findStateUpdateUsingSymbol(child, sym); found != nil {
			return found
		}
	}
	return nil
}

func usesSymbol(node *reader.RichNode, sym string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol && node.Value == sym {
		return true
	}
	for _, child := range node.Children {
		if usesSymbol(child, sym) {
			return true
		}
	}
	return false
}

func init() {
	defaultRule := &NestedAtomsRule{
		Rule: rules.Rule{
			ID:          "nested-atoms",
			Name:        "Nested Atoms",
			Description: "Detects an Atom or other managed reference (like a Volatile or Ref) inside another Atom",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
