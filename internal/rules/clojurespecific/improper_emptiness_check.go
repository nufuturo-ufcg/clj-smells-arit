package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
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

func isResolvedSafeCollectionProducer(node *reader.RichNode, facts semantics.Facts) bool {
	if node == nil || node.Type != reader.NodeList || !facts.ResolutionKnown {
		return false
	}
	canonicalName := semantics.CallName(node)
	if !validSafeCollectionProducerArity(node, canonicalName) {
		return false
	}
	switch canonicalName {
	case "clojure.core/hash-map", "clojure.core/keys", "clojure.core/list",
		"clojure.core/set", "clojure.core/sort", "clojure.core/sort-by",
		"clojure.core/str", "clojure.core/vals", "clojure.core/vector":
		return true
	default:
		return false
	}
}

func validSafeCollectionProducerArity(node *reader.RichNode, canonicalName string) bool {
	if node == nil {
		return false
	}
	facts := semantics.ForNode(node, nil)
	return facts.ResolutionKnown && facts.ArityKnown && facts.ArityValid && semantics.CallName(node) == canonicalName
}

func isInvalidResolvedCollectionProducer(node *reader.RichNode, context map[string]interface{}) bool {
	if node == nil || node.Type != reader.NodeList {
		return false
	}
	facts := rules.FactsForNode(node, context)
	if !facts.ResolutionKnown || !facts.ArityKnown {
		return false
	}
	if !facts.ArityValid {
		return true
	}
	switch semantics.CallName(node) {
	case "clojure.core/hash-map", "clojure.core/keys", "clojure.core/vals",
		"clojure.core/list", "clojure.core/set", "clojure.core/sort",
		"clojure.core/sort-by", "clojure.core/str", "clojure.core/vector":
		return !validSafeCollectionProducerArity(node, semantics.CallName(node))
	default:
		return false
	}
}

func knownCollectionForEmptiness(node *reader.RichNode, context map[string]interface{}) bool {
	if node == nil {
		return false
	}
	facts := rules.FactsForNode(node, context)
	if node.Type == reader.NodeList && facts.ArityKnown && !facts.ArityValid {
		return false
	}
	if node.Type == reader.NodeList {
		if (facts.Type == semantics.TypeVector || facts.Type == semantics.TypeMap ||
			facts.Type == semantics.TypeSet || facts.Type == semantics.TypeString) &&
			rules.HasProvenTypeFact(facts) ||
			isResolvedSafeCollectionProducer(node, *facts) {
			return true
		}
		if summaries, ok := context["function-summaries"].(map[string]semantics.FunctionSummary); ok {
			candidates := []string{semantics.CallName(node)}
			if len(node.Children) > 0 && node.Children[0] != nil {
				candidates = append(candidates, node.Children[0].Value)
			}
			for _, candidate := range candidates {
				if summary, found := summaries[candidate]; found && summary.Evidence == semantics.EvidenceProven {
					switch summary.ReturnsType {
					case semantics.TypeString, semantics.TypeList, semantics.TypeVector, semantics.TypeMap, semantics.TypeSet, semantics.TypeNil:
						return true
					}
				}
			}
		}
		return false
	}
	switch facts.Type {
	case semantics.TypeNil, semantics.TypeString, semantics.TypeList,
		semantics.TypeVector, semantics.TypeMap, semantics.TypeSet:
		return rules.HasProvenTypeFact(facts)
	default:
		return false
	}
}

func (r *ImproperEmptinessCheckRule) finding(node *reader.RichNode, filepath, message string, contextual bool) *rules.Finding {
	finding := &rules.Finding{
		RuleID: r.ID, Message: message, Filepath: filepath,
		Location: node.Location, Severity: r.Severity,
	}
	if contextual {
		return rules.SetContextualFindingWithEvidence(finding, "The replacement may depend on whether the result is consumed as a boolean or as a collection value.", "return-contract", "collection-type")
	}
	return finding
}

func (r *ImproperEmptinessCheckRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || (node.Type != reader.NodeList && node.Type != reader.NodeFnLiteral) || len(node.Children) < 2 {
		return nil
	}

	// when-not/if-not consume truthiness themselves, so this rewrite is exact.
	if isCoreCall(node, "when-not", "if-not") {
		arg := node.Children[1]
		if isCoreCall(arg, "empty?") && len(arg.Children) == 2 {
			if !isCoreReplacementAvailable(context, "seq") {
				return nil
			}
			collection := getVerboseNodeText(arg.Children[1])
			replacement := "when"
			if isCoreCall(node, "if-not") {
				replacement = "if"
			}
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: `(%s (empty? %s))`. Consider using `(%s (seq %s) ...)`.",
				node.Children[0].Value, collection, replacement, collection), false)
		}
	}

	if isCoreCall(node, "not") && len(node.Children) == 2 {
		arg := node.Children[1]
		if isCoreCall(arg, "empty?") && len(arg.Children) == 2 {
			collection := getVerboseNodeText(arg.Children[1])
			if !usedOnlyForTruthiness(node, context) {
				if !isCoreReplacementAvailable(context, "seq") && !isCoreReplacementAvailable(context, "boolean") {
					return nil
				}
				return r.finding(node, filepath, fmt.Sprintf(
					"Improper emptiness check: `(not (empty? %s))` occurs where the return contract is unknown. Review whether `(seq %s)` or `(boolean (seq %s))` preserves the public value contract.", collection, collection, collection), true)
			}
			if !isCoreReplacementAvailable(context, "seq") {
				return nil
			}
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: `(not (empty? %s))`. Consider using `(seq %s)` or `(boolean (seq %s))`.", collection, collection, collection), false)
		}
	}

	if isCoreCall(node, "zero?", "pos?") && len(node.Children) == 2 {
		collectionNode, ok := countArgument(node.Children[1])
		if ok {
			if isInvalidResolvedCollectionProducer(collectionNode, context) {
				return nil
			}
			collection := getVerboseNodeText(collectionNode)
			if isCoreCall(node, "zero?") {
				if !isCoreReplacementAvailable(context, "empty?") {
					return nil
				}
				if knownCollectionForEmptiness(collectionNode, context) {
					return r.finding(node, filepath, fmt.Sprintf(
						"Improper emptiness check: `(zero? (count %s))`. Consider using `(empty? %s)`.", collection, collection), false)
				}
				return r.finding(node, filepath, fmt.Sprintf(
					"Improper emptiness check: `(zero? (count %s))`. Consider using `(empty? %s)`.", collection, collection), true)
			}
			if !isCoreReplacementAvailable(context, "seq") {
				return nil
			}
			if knownCollectionForEmptiness(collectionNode, context) && usedOnlyForTruthiness(node, context) {
				return r.finding(node, filepath, fmt.Sprintf(
					"Improper emptiness check: `(pos? (count %s))`. Consider using `(seq %s)`.", collection, collection), false)
			}
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: `(pos? (count %s))`. Consider using `(seq %s)`.", collection, collection), true)
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
	if isInvalidResolvedCollectionProducer(collectionNode, context) {
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
		if !isCoreReplacementAvailable(context, "empty?") {
			return nil
		}
		if knownCollectionForEmptiness(collectionNode, context) {
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: using `%s` with `count`. Consider using `(empty? %s)`.", op, collection), false)
		}
		return r.finding(node, filepath, fmt.Sprintf(
			"Improper emptiness check: using `%s` with `count`. Consider using `(empty? %s)`.", op, collection), true)
	}
	if nonEmptyCheck {
		if !isCoreReplacementAvailable(context, "seq") {
			return nil
		}
		if knownCollectionForEmptiness(collectionNode, context) && usedOnlyForTruthiness(node, context) {
			return r.finding(node, filepath, fmt.Sprintf(
				"Improper emptiness check: using `%s` with `count`. Consider using `(seq %s)`.", op, collection), false)
		}
		return r.finding(node, filepath, fmt.Sprintf(
			"Improper emptiness check: using `%s` with `count`. Consider using `(seq %s)` or `(not (empty? %s))`.", op, collection, collection), true)
	}
	return nil
}

func init() {
	rules.RegisterRule(&ImproperEmptinessCheckRule{Rule: rules.Rule{
		ID: "improper-emptiness-check", Name: "Improper Emptiness Check",
		Description:           "Detects semantics-preserving opportunities to replace verbose collection emptiness checks.",
		ContextualDescription: "It may be contextual when the function exposes a boolean, when seq would change the return type, or when the collection contract is unknown. The finding remains visible with --include-contextual.",
		Severity:              rules.SeverityHint,
	}})
}
