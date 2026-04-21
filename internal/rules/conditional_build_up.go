package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

const minConditionalRebinds = 1

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

	switch node.Value {
	case "let", "clojure.core/let", "cljs.core/let":
		return true
	default:
		return false
	}
}

func isSimpleLocalSymbol(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeSymbol {
		return false
	}

	v := node.Value
	if v == "" || v == "_" || v == "&" {
		return false
	}

	if strings.Contains(v, "/") {
		return false
	}

	return true
}

func isIfAssocRebindPattern(valueNode *reader.RichNode, sym string) bool {
	if valueNode == nil || valueNode.Type != reader.NodeList || len(valueNode.Children) != 4 {
		return false
	}

	head := valueNode.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return false
	}

	ifName := head.Value
	if ifName != "if" && ifName != "clojure.core/if" && ifName != "cljs.core/if" {
		return false
	}

	thenBranch := valueNode.Children[2]
	elseBranch := valueNode.Children[3]

	if elseBranch == nil || elseBranch.Type != reader.NodeSymbol || elseBranch.Value != sym {
		return false
	}

	if thenBranch == nil || thenBranch.Type != reader.NodeList || len(thenBranch.Children) < 4 {
		return false
	}

	assocSym := thenBranch.Children[0]
	if assocSym == nil || assocSym.Type != reader.NodeSymbol {
		return false
	}

	assocName := assocSym.Value
	if assocName != "assoc" &&
		assocName != "clojure.core/assoc" &&
		assocName != "cljs.core/assoc" &&
		!strings.HasSuffix(assocName, "/assoc") {
		return false
	}

	firstArg := thenBranch.Children[1]
	return firstArg != nil &&
		firstArg.Type == reader.NodeSymbol &&
		firstArg.Value == sym
}

func referencesSymbol(node *reader.RichNode, sym string) bool {
	if node == nil {
		return false
	}

	if node.Type == reader.NodeSymbol && node.Value == sym {
		return true
	}

	for _, child := range node.Children {
		if referencesSymbol(child, sym) {
			return true
		}
	}

	return false
}

func isBaseBindingCandidate(valueNode *reader.RichNode, sym string) bool {
	if valueNode == nil {
		return false
	}

	if isIfAssocRebindPattern(valueNode, sym) {
		return false
	}

	if referencesSymbol(valueNode, sym) {
		return false
	}

	return true
}

func makeConditionalBuildUpFinding(r Rule, letNode *reader.RichNode, filepath, name string, count int) *Finding {
	updateWord := "updates"
	if count == 1 {
		updateWord = "update"
	}

	return &Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Same symbol '%s' is rebound with %d successive conditional %s using `(if ... (assoc %s ...) %s)`. Prefer `cond->` for clarity.",
			name, count, updateWord, name, name,
		),
		Filepath: filepath,
		Location: letNode.Location,
		Severity: r.Severity,
	}
}

func minBindingsChildren() int {
	return 2 * (1 + minConditionalRebinds)
}

func (r *ConditionalBuildupRule) detectLetConditionalBuildUp(letNode *reader.RichNode, filepath string) *Finding {
	if len(letNode.Children) < 3 {
		return nil
	}

	bindingsNode := letNode.Children[1]
	if bindingsNode == nil || bindingsNode.Type != reader.NodeVector || len(bindingsNode.Children) < minBindingsChildren() {
		return nil
	}

	currentName := ""
	hasBaseBinding := false
	conditionalRebindCount := 0

	bestName := ""
	bestCount := 0

	resetRun := func() {
		currentName = ""
		hasBaseBinding = false
		conditionalRebindCount = 0
	}

	startRun := func(name string, valueNode *reader.RichNode) {
		currentName = name
		conditionalRebindCount = 0

		hasBaseBinding = isBaseBindingCandidate(valueNode, name)
	}

	for i := 0; i+1 < len(bindingsNode.Children); i += 2 {
		nameNode := bindingsNode.Children[i]
		valueNode := bindingsNode.Children[i+1]

		if !isSimpleLocalSymbol(nameNode) || valueNode == nil {
			resetRun()
			continue
		}

		name := nameNode.Value

		if currentName == "" || name != currentName {
			startRun(name, valueNode)
			continue
		}

		if hasBaseBinding && isIfAssocRebindPattern(valueNode, name) {
			conditionalRebindCount++

			if conditionalRebindCount > bestCount {
				bestCount = conditionalRebindCount
				bestName = name
			}

			continue
		}

		startRun(name, valueNode)
	}

	if bestCount >= minConditionalRebinds {
		return makeConditionalBuildUpFinding(r.Meta(), letNode, filepath, bestName, bestCount)
	}

	return nil
}

func (r *ConditionalBuildupRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild == nil || firstChild.Type != reader.NodeSymbol {
		return nil
	}

	if !isLetForm(firstChild) {
		return nil
	}

	return r.detectLetConditionalBuildUp(node, filepath)
}

func init() {
	defaultRule := &ConditionalBuildupRule{
		Rule: Rule{
			ID:          "conditional-build-up",
			Name:        "Conditional Build-Up",
			Description: "Detects contiguous let rebindings where the same simple symbol is rebuilt through successive conditional assoc updates. Prefer cond-> when conditional map updates are chained.",
			Severity:    SeverityHint,
		},
	}

	RegisterRule(defaultRule)
}