package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type NestedAtomsRule struct {
	rules.Rule
}

func (r *NestedAtomsRule) Meta() rules.Rule { return r.Rule }

var stateCreationNames = []string{
	"clojure.core/atom",
	"clojure.core/ref",
	"clojure.core/agent",
	"clojure.core/volatile!",
}

func isStateCreation(node *reader.RichNode) bool {
	return rules.CallResolvesTo(node, stateCreationNames...)
}

// A nested reference is certain when it is part of a literal value. Descending
// through arbitrary calls would assume what an unknown function returns and was
// the source of most real-world false positives.
func literalContainsStateCreation(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if isStateCreation(node) {
		return true
	}
	switch node.Type {
	case reader.NodeMap, reader.NodeVector, reader.NodeSet:
		for _, child := range node.Children {
			if literalContainsStateCreation(child) {
				return true
			}
		}
	}
	return false
}

func rawSymbolOccurs(node *reader.RichNode, symbol string) bool {
	if node == nil {
		return false
	}
	// @inner inserts inner's value, not the reference itself.
	switch node.Type {
	case reader.NodeDeref, reader.NodeQuote, reader.NodeSyntaxQuote,
		reader.NodeVarQuote, reader.NodeReaderDiscard:
		return false
	case reader.NodeSymbol:
		return node.Value == symbol
	}
	for _, child := range node.Children {
		if rawSymbolOccurs(child, symbol) {
			return true
		}
	}
	return false
}

func updateCertainlyStores(node *reader.RichNode, innerSymbol string) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}

	switch {
	case rules.CallResolvesTo(node, "clojure.core/reset!", "clojure.core/vreset!", "clojure.core/ref-set"):
		return rawSymbolOccurs(node.Children[2], innerSymbol)

	case rules.CallResolvesTo(node, "clojure.core/swap!", "clojure.core/vswap!", "clojure.core/alter", "clojure.core/commute"):
		updater := node.Children[2]
		if updater.Type != reader.NodeSymbol || updater.Resolution == nil {
			return false
		}
		switch updater.Resolution.CanonicalName {
		case "clojure.core/conj", "clojure.core/assoc", "clojure.core/assoc-in":
			for _, arg := range node.Children[3:] {
				if rawSymbolOccurs(arg, innerSymbol) {
					return true
				}
			}
		}
	}
	return false
}

func findCertainInsertion(node *reader.RichNode, innerSymbol string) bool {
	if node == nil {
		return false
	}
	if updateCertainlyStores(node, innerSymbol) {
		return true
	}
	// Definitions and unevaluated forms do not execute as part of this body.
	if node.Type == reader.NodeQuote || node.Type == reader.NodeSyntaxQuote ||
		node.Type == reader.NodeVarQuote || node.Type == reader.NodeReaderDiscard {
		return false
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		switch node.Children[0].Value {
		case "fn", "fn*", "defn", "defn-", "defmacro", "delay", "lazy-seq":
			return false
		}
	}
	for _, child := range node.Children {
		if findCertainInsertion(child, innerSymbol) {
			return true
		}
	}
	return false
}

func (r *NestedAtomsRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	if isStateCreation(node) {
		for _, initialValue := range node.Children[1:] {
			if literalContainsStateCreation(initialValue) {
				return &rules.Finding{
					RuleID: r.ID, Message: "Found nested Atom/Ref/Volatile/Agent inside a stateful reference.",
					Filepath: filepath, Location: node.Location, Severity: r.Severity,
				}
			}
		}
	}

	// Track a freshly-created local reference only when a recognized state
	// update certainly stores the reference itself. Merely calling swap! on the
	// same local atom is not nesting.
	if rules.CallResolvesTo(node, "clojure.core/let", "clojure.core/let*") && len(node.Children) >= 3 {
		bindings := node.Children[1]
		if bindings.Type == reader.NodeVector {
			for i := 0; i+1 < len(bindings.Children); i += 2 {
				symbol, value := bindings.Children[i], bindings.Children[i+1]
				if symbol.Type != reader.NodeSymbol || !isStateCreation(value) {
					continue
				}
				for _, body := range node.Children[2:] {
					if findCertainInsertion(body, symbol.Value) {
						return &rules.Finding{
							RuleID:   r.ID,
							Message:  "Found stateful reference created in let binding being inserted into another stateful reference.",
							Filepath: filepath, Location: node.Location, Severity: r.Severity,
						}
					}
				}
			}
		}
	}

	if rules.CallResolvesTo(node, "clojure.core/swap!", "clojure.core/vswap!", "clojure.core/alter", "clojure.core/commute", "clojure.core/reset!", "clojure.core/vreset!", "clojure.core/ref-set") && len(node.Children) >= 3 {
		for _, arg := range node.Children[2:] {
			if containsAnyStateCreation(arg) {
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  "Found stateful reference being created and inserted into another stateful reference.",
					Filepath: filepath, Location: node.Location, Severity: r.Severity,
				}
			}
		}
	}

	return nil
}

func containsAnyStateCreation(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if isStateCreation(node) {
		return true
	}
	for _, child := range node.Children {
		if containsAnyStateCreation(child) {
			return true
		}
	}
	return false
}

func init() {
	rules.RegisterRule(&NestedAtomsRule{Rule: rules.Rule{
		ID: "nested-atoms", Name: "Nested Atoms",
		Description: "Detects an Atom or other managed reference (like a Volatile or Ref) inside another Atom",
		Severity:    rules.SeverityWarning,
	}})
}
