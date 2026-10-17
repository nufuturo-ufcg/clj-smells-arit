package semantics

import (
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type FunctionSummary struct {
	Name          string
	QualifiedName string
	Effects       Effect
	EagerEffects  Effect
	ReturnsType   AbstractType
	ReturnsLazy   bool
	ReturnsBool   bool
	UsesDynamic   bool
	Evidence      Evidence
	EagerEvidence Evidence
	// GuardedLoad is true when the function loads a namespace inside a try
	// with a catch: a probe for an optional dependency, not a hard load.
	GuardedLoad bool
}

func BuildFunctionSummaries(roots []*reader.RichNode, namespace string) map[string]FunctionSummary {
	summaries := make(map[string]FunctionSummary)
	for _, root := range roots {
		collectFunctionSummaries(root, namespace, summaries, false)
	}
	return summaries
}

// collectFunctionSummaries records defn summaries. A definition inside a
// reader conditional exists only on some platforms, and another branch may
// define the same name differently (e.g. a defmacro that runs at compile
// time), so its evidence is downgraded to unknown for the file as a whole.
func collectFunctionSummaries(node *reader.RichNode, namespace string, summaries map[string]FunctionSummary, platformSpecific bool) {
	if node == nil {
		return
	}
	if node.Type == reader.NodeReaderCond || node.Type == reader.NodeReaderCondSplice {
		platformSpecific = true
	}
	if node.Type == reader.NodeList && len(node.Children) > 1 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
		head := node.Children[0].Value
		if head == "defn" || head == "defn-" {
			if node.Children[1] != nil && node.Children[1].Type == reader.NodeSymbol {
				name := node.Children[1].Value
				summary := summarizeFunction(node, name, namespace)
				if platformSpecific {
					summary.Evidence = EvidenceUnknown
					summary.EagerEvidence = EvidenceUnknown
				}
				summaries[name] = summary
				if summary.QualifiedName != "" {
					summaries[summary.QualifiedName] = summary
				}
			}
		}
	}
	for _, child := range node.Children {
		collectFunctionSummaries(child, namespace, summaries, platformSpecific)
	}
}

func summarizeFunction(node *reader.RichNode, name, namespace string) FunctionSummary {
	summary := FunctionSummary{
		Name:          name,
		QualifiedName: name,
		ReturnsType:   TypeUnknown,
		Evidence:      EvidenceProven,
		EagerEvidence: EvidenceProven,
	}
	if namespace != "" {
		summary.QualifiedName = namespace + "/" + name
	}

	bodyStart := functionBodyStart(node)
	if bodyStart >= len(node.Children) {
		return summary
	}
	for _, body := range node.Children[bodyStart:] {
		if containsGuardedNamespaceLoad(body, false) {
			summary.GuardedLoad = true
		}
	}
	for index, body := range node.Children[bodyStart:] {
		lastFacts := accumulateSummaryAndEager(body, &summary, true)
		if index == len(node.Children[bodyStart:])-1 && body != nil {
			summary.ReturnsType = summaryReturnType(body, lastFacts)
			summary.ReturnsLazy = lastFacts.Laziness == Lazy
			summary.ReturnsBool = returnsBoolean(body)
		}
	}
	return summary
}

// summaryReturnType keeps a value type only. Any call form has the syntactic
// type list; it describes the returned value only when a recognized producer
// such as map or filter established its laziness. An unknown call, or a let/if
// whose value is not analyzed, returns a value of unknown type.
func summaryReturnType(body *reader.RichNode, facts Facts) AbstractType {
	if facts.Type == TypeList && (body.Type == reader.NodeList || body.Type == reader.NodeFnLiteral) &&
		facts.Laziness == LazinessUnknown {
		return TypeUnknown
	}
	return facts.Type
}

// accumulateSummaryAndEager collects both the broad potential effects and the
// effects executed while entering a function in one traversal. The broad
// summary always descends into children; the eager summary stops at deferred
// boundaries, matching the former two-pass implementation.
func accumulateSummaryAndEager(node *reader.RichNode, summary *FunctionSummary, eager bool) Facts {
	if node == nil || summary == nil || isDeferredSummaryNode(node) {
		if node == nil || summary == nil {
			return Facts{}
		}
	}

	facts := ForNode(node, nil)
	summary.Effects |= facts.Effects
	if facts.Effects.Has(EffectUnknown) {
		summary.Evidence = EvidenceUnknown
	}
	if node.Type == reader.NodeSymbol && strings.HasPrefix(node.Value, "*") && strings.HasSuffix(node.Value, "*") {
		summary.UsesDynamic = true
	}

	deferred := isDeferredSummaryNode(node)
	if eager && !deferred {
		summary.EagerEffects |= facts.Effects
		if facts.Effects.Has(EffectUnknown) {
			summary.EagerEvidence = EvidenceUnknown
		}
	}
	for _, child := range node.Children {
		accumulateSummaryAndEager(child, summary, eager && !deferred)
	}
	return facts
}

func isDeferredSummaryNode(node *reader.RichNode) bool {
	if node == nil {
		return true
	}
	if node.Type == reader.NodeFnLiteral || node.Type == reader.NodeQuote ||
		node.Type == reader.NodeSyntaxQuote || node.Type == reader.NodeVarQuote ||
		node.Type == reader.NodeReaderDiscard {
		return true
	}
	if node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0] == nil ||
		node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	switch node.Children[0].Value {
	case "defn", "defn-", "defmacro", "fn", "fn*", "letfn", "deftest",
		"delay", "lazy-seq", "future", "future-call":
		return true
	default:
		return false
	}
}

func functionBodyStart(node *reader.RichNode) int {
	if node == nil || len(node.Children) < 3 {
		return len(node.Children)
	}
	index := 2
	if node.Children[index] != nil && node.Children[index].Type == reader.NodeString {
		index++
	}
	if index < len(node.Children) && node.Children[index] != nil && node.Children[index].Type == reader.NodeMap {
		index++
	}
	if index < len(node.Children) && node.Children[index] != nil && node.Children[index].Type == reader.NodeVector {
		index++
	}
	return index
}

func returnsBoolean(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0] == nil {
		return false
	}
	if node.Children[0].Resolution != nil {
		switch node.Children[0].Resolution.CanonicalName {
		case "clojure.core/not", "clojure.core/boolean", "clojure.core/true?", "clojure.core/false?",
			"clojure.core/zero?", "clojure.core/pos?", "clojure.core/neg?", "clojure.core/empty?":
			return true
		}
	}
	switch node.Children[0].Value {
	case "=", "==", "not=", ">", "<", ">=", "<=", "and", "or":
		return true
	default:
		return false
	}
}

func containsGuardedNamespaceLoad(node *reader.RichNode, guarded bool) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
		head := strings.TrimPrefix(node.Children[0].Value, "clojure.core/")
		switch head {
		case "require", "use", "requiring-resolve", "load", "load-file":
			if guarded {
				return true
			}
		case "try":
			for _, child := range node.Children[1:] {
				if child != nil && child.Type == reader.NodeList && len(child.Children) > 0 &&
					child.Children[0] != nil && child.Children[0].Value == "catch" {
					guarded = true
					break
				}
			}
		}
	}
	for _, child := range node.Children {
		if containsGuardedNamespaceLoad(child, guarded) {
			return true
		}
	}
	return false
}
