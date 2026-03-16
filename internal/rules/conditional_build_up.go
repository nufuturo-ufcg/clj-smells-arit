package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type ConditionalBuildupRule struct {
	Rule
}

func (r *ConditionalBuildupRule) Meta() Rule {
	return r.Rule
}

func isLetForm(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeSymbol {
		return false
	}

	v := node.Value
	return v == "let" || (strings.HasSuffix(v, "/let") && v != "if-let" && v != "when-let")
}

func isIfAssocRebindPattern(valueNode *reader.RichNode, sym string) bool {
	if valueNode == nil || valueNode.Type != reader.NodeList || len(valueNode.Children) < 4 {
		return false
	}

	head := valueNode.Children[0]
	if head.Type != reader.NodeSymbol || head.Value != "if" {
		return false
	}

	thenBranch := valueNode.Children[2]
	elseBranch := valueNode.Children[3]
	if elseBranch.Type != reader.NodeSymbol || elseBranch.Value != sym {
		return false
	}
	if thenBranch.Type != reader.NodeList || len(thenBranch.Children) < 3 {
		return false
	}

	assocSym := thenBranch.Children[0]
	if assocSym.Type != reader.NodeSymbol {
		return false
	}

	assocName := assocSym.Value
	if assocName != "assoc" && !strings.HasSuffix(assocName, "/assoc") {
		return false
	}

	firstArg := thenBranch.Children[1]
	return firstArg.Type == reader.NodeSymbol && firstArg.Value == sym
}

func (r *ConditionalBuildupRule) detectLetConditionalBuildUp(letNode *reader.RichNode) *Finding {
	if len(letNode.Children) < 2 {
		return nil
	}

	bindingsNode := letNode.Children[1]
	if bindingsNode.Type != reader.NodeVector || len(bindingsNode.Children) < 4 {
		return nil
	}

	nameToValues := make(map[string][]*reader.RichNode)
	var nameOrder []string
	seen := make(map[string]bool)
	for i := 0; i+1 < len(bindingsNode.Children); i += 2 {
		nameNode := bindingsNode.Children[i]
		valueNode := bindingsNode.Children[i+1]
		if nameNode == nil || valueNode == nil {
			continue
		}
		if nameNode.Type != reader.NodeSymbol {
			continue
		}

		name := nameNode.Value
		if name == "_" || name == "&" {
			continue
		}
		
		nameToValues[name] = append(nameToValues[name], valueNode)
		if !seen[name] {
			seen[name] = true
			nameOrder = append(nameOrder, name)
		}
	}

	for _, name := range nameOrder {
		values := nameToValues[name]
		if len(values) < 2 {
			continue
		}

		var assocRebindCount int
		for _, val := range values {
			if isIfAssocRebindPattern(val, name) {
				assocRebindCount++
			}
		}
		
		if assocRebindCount < 1 {
			continue
		}

		meta := r.Meta()
		return &Finding{
			RuleID:   meta.ID,
			Message: fmt.Sprintf("Same symbol '%s' is rebound with at least one conditional update (if pred (assoc %s ...) %s). Prefer building the map with cond-> for clarity: (cond-> initial pred1 (assoc :k1 v1) pred2 (assoc :k2 v2) ...)", name, name, name),
			Filepath: "",
			Location: letNode.Location,
			Severity: meta.Severity,
		}
	}
	return nil
}

func (r *ConditionalBuildupRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return nil
	}

	if isLetForm(firstChild) {
		if finding := r.detectLetConditionalBuildUp(node); finding != nil {
			finding.Filepath = filepath
			return finding
		}
		return nil
	}

	return nil
}

func init() {
	defaultRule := &ConditionalBuildupRule{
		Rule: Rule{
			ID:   "conditional-build-up",
			Name: "Conditional Build-Up",
			Description: "Detects let bindings where the same symbol is rebound multiple times using conditional assoc patterns like (if pred (assoc m k v) m). Prefer cond-> to express these conditional updates declaratively. Based on bsless.github.io/code-smells.",
			Severity:    SeverityHint,
		},
	}

	RegisterRule(defaultRule)
}
