package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ImproperEmptinessCheckRule struct{ rules.Rule }

func (r *ImproperEmptinessCheckRule) Meta() rules.Rule { return r.Rule }

func isCoreCall(node *reader.RichNode, names ...string) bool {
	canonical := make([]string, 0, len(names))
	for _, name := range names {
		canonical = append(canonical, "clojure.core/"+name)
	}
	return rules.CallResolvesTo(node, canonical...)
}

func directChildIndex(parent, node *reader.RichNode) int {
	if parent == nil {
		return -1
	}
	for i, child := range parent.Children {
		if child == node {
			return i
		}
	}
	return -1
}

// seq is a truthy/falsey predicate but does not return a boolean for a
// non-empty collection. Recommend it only where Clojure consumes truthiness;
// replacing a public function's boolean return value with a sequence is not
// semantics-preserving.
func usedOnlyForTruthiness(node *reader.RichNode, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) == 0 {
		return false
	}
	parent := ancestors[len(ancestors)-1]
	idx := directChildIndex(parent, node)
	if idx < 1 {
		return false
	}

	switch {
	case isCoreCall(parent, "if", "if-not", "when", "when-not", "while"):
		return idx == 1
	case isCoreCall(parent, "not", "boolean"):
		return idx == 1
	case isCoreCall(parent, "and", "or"):
		return idx < len(parent.Children)-1
	case isCoreCall(parent, "cond"):
		return idx%2 == 1
	}
	return false
}

func countArgument(node *reader.RichNode) (*reader.RichNode, bool) {
	if !isCoreCall(node, "count") || len(node.Children) != 2 {
		return nil, false
	}
	return node.Children[1], true
}

func numberIs(node *reader.RichNode, value string) bool {
	return node != nil && node.Type == reader.NodeNumber && node.Value == value
}

func (r *ImproperEmptinessCheckRule) finding(node *reader.RichNode, filepath, message string) *rules.Finding {
	return &rules.Finding{
		RuleID: r.ID, Message: message, Filepath: filepath,
		Location: node.Location, Severity: r.Severity,
	}
}

func (r *ImproperEmptinessCheckRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || (node.Type != reader.NodeList && node.Type != reader.NodeFnLiteral) || len(node.Children) < 2 {
		return nil
	}

	// when-not/if-not consume truthiness themselves, so this rewrite is exact.
	if isCoreCall(node, "when-not", "if-not") {
		arg := node.Children[1]
		if isCoreCall(arg, "empty?") && len(arg.Children) == 2 {
			collection := getVerboseNodeText(arg.Children[1])
			replacement := "when"
			if isCoreCall(node, "if-not") {
				replacement = "if"
			}
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: `(%s (empty? %s))`. Consider using `(%s (seq %s) ...)`.",
				node.Children[0].Value, collection, replacement, collection))
		}
	}

	if isCoreCall(node, "not") && len(node.Children) == 2 {
		arg := node.Children[1]
		if isCoreCall(arg, "empty?") && len(arg.Children) == 2 {
			collection := getVerboseNodeText(arg.Children[1])
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: `(not (empty? %s))`. Consider using `(seq %s)` or `(boolean (seq %s))`.", collection, collection, collection))
		}
	}

	if isCoreCall(node, "zero?", "pos?") && len(node.Children) == 2 {
		collectionNode, ok := countArgument(node.Children[1])
		if ok {
			collection := getVerboseNodeText(collectionNode)
			if isCoreCall(node, "zero?") {
				return r.finding(node, filepath, fmt.Sprintf(
					"Improper emptiness check: `(zero? (count %s))`. Consider using `(empty? %s)`.", collection, collection))
			}
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: `(pos? (count %s))`. Consider using `(seq %s)`.", collection, collection))
		}
	}

	if len(node.Children) != 3 || !isCoreCall(node, "=", "==", "not=", ">", "<", ">=", "<=") {
		return nil
	}
	op := node.Children[0].Value
	left, right := node.Children[1], node.Children[2]
	countNode, constant, countOnLeft := left, right, true
	collectionNode, ok := countArgument(countNode)
	if !ok {
		countNode, constant, countOnLeft = right, left, false
		collectionNode, ok = countArgument(countNode)
	}
	if !ok {
		return nil
	}

	emptyCheck := numberIs(constant, "0") && (op == "=" || op == "==")
	nonEmptyCheck := false
	if numberIs(constant, "0") {
		nonEmptyCheck = op == "not=" || (countOnLeft && op == ">") || (!countOnLeft && op == "<")
	}
	if numberIs(constant, "1") {
		nonEmptyCheck = (countOnLeft && op == ">=") || (!countOnLeft && op == "<=")
	}
	collection := getVerboseNodeText(collectionNode)
	if emptyCheck {
		return r.finding(node, filepath, fmt.Sprintf(
			"Improper emptiness check: using `%s` with `count`. Consider using `(empty? %s)`.", op, collection))
	}
	if nonEmptyCheck {
		return r.finding(node, filepath, fmt.Sprintf(
			"Improper emptiness check: using `%s` with `count`. Consider using `(seq %s)` or `(not (empty? %s))`.", op, collection, collection))
	}
	return nil
}

func init() {
	rules.RegisterRule(&ImproperEmptinessCheckRule{Rule: rules.Rule{
		ID: "improper-emptiness-check", Name: "Improper Emptiness Check",
		Description: "Detects semantics-preserving opportunities to replace verbose collection emptiness checks.",
		Severity:    rules.SeverityHint,
	}})
}
