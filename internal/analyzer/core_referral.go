package analyzer

import (
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

// coreReferral records how the namespace refers clojure.core, so that symbol
// resolution never claims clojure.core for a name the ns declaration removed,
// and rules can tell when an unqualified core name may have been replaced.
type coreReferral struct {
	excluded map[string]bool
	only     map[string]bool // nil when :only is absent
	// ambiguous is true when a whole namespace is referred (:use, :refer
	// :all, or a top-level use/refer call). Clojure then replaces any
	// clojure.core mapping with a same-named var of that namespace and only
	// prints a warning, so no unqualified core name is provably core.
	ambiguous bool
}

// Special forms are not vars: :refer-clojure cannot remove them.
var nonExcludableSpecialForms = map[string]bool{
	"def": true, "if": true, "do": true, "let*": true, "fn*": true, "loop*": true, "recur": true,
	"quote": true, "var": true, "throw": true, "try": true, "catch": true, "finally": true,
	"monitor-enter": true, "monitor-exit": true, "new": true, "set!": true, ".": true,
	"letfn*": true, "case*": true, "import*": true, "deftype*": true, "reify*": true,
}

func (c *coreReferral) removes(name string) bool {
	if c == nil || nonExcludableSpecialForms[name] {
		return false
	}
	if c.excluded[name] {
		return true
	}
	return c.only != nil && !c.only[name]
}

func collectCoreReferral(nsNode *reader.RichNode, roots []*reader.RichNode) *coreReferral {
	referral := &coreReferral{excluded: map[string]bool{}}
	if nsNode != nil {
		for _, clause := range nsNode.Children[1:] {
			if clause == nil || clause.Type != reader.NodeList || len(clause.Children) == 0 {
				continue
			}
			switch clause.Children[0].Value {
			case ":refer-clojure":
				referral.readReferClojure(clause.Children[1:])
			case ":use":
				for _, spec := range clause.Children[1:] {
					referral.referWholeLibspec(spec, false)
				}
			case ":require":
				for _, spec := range clause.Children[1:] {
					if containsReferAll(spec) {
						referral.referWholeLibspec(spec, true)
					}
				}
			}
		}
	}
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) == 0 || root.Children[0].Type != reader.NodeSymbol {
			continue
		}
		switch root.Children[0].Value {
		case "use", "clojure.core/use", "refer", "clojure.core/refer":
			referral.ambiguous = true
		case "require", "clojure.core/require":
			if containsReferAll(root) {
				referral.ambiguous = true
			}
		case "refer-clojure", "clojure.core/refer-clojure":
			referral.readReferClojure(root.Children[1:])
		}
	}
	return referral
}

// knownNamespaceCoreCollisions lists, for namespaces shipped with Clojure
// 1.11.1, the public vars whose names collide with clojure.core (computed with
// ns-publics; evidence in the correction report). Referring such a namespace
// whole replaces exactly these names; an empty list means no replacement.
var knownNamespaceCoreCollisions = map[string][]string{
	"clojure.test": nil, "clojure.set": nil, "clojure.walk": nil, "clojure.pprint": nil,
	"clojure.java.io": nil, "clojure.data": nil, "clojure.repl": nil, "clojure.template": nil,
	"clojure.xml": nil, "clojure.java.shell": nil, "clojure.stacktrace": nil, "clojure.inspector": nil,
	"clojure.reflect": nil, "clojure.instant": nil, "clojure.datafy": nil, "clojure.test.junit": nil,
	"clojure.core.protocols": nil, "clojure.java.javadoc": nil, "clojure.java.browse": nil,
	"clojure.string":        {"replace", "reverse"},
	"clojure.edn":           {"read", "read-string"},
	"clojure.zip":           {"next", "remove", "replace"},
	"clojure.main":          {"with-bindings"},
	"clojure.spec.alpha":    {"*", "+", "and", "assert", "cat", "keys", "merge", "or"},
	"clojure.core.reducers": {"cat", "drop", "filter", "flatten", "map", "mapcat", "reduce", "remove", "take", "take-while"},
}

// referWholeLibspec records a libspec that refers every var of a namespace.
// A libspec limited by :only refers just the listed names, which then shadow
// clojure.core. Unknown namespaces and prefix lists make core names ambiguous.
func (c *coreReferral) referWholeLibspec(spec *reader.RichNode, referAll bool) {
	if spec == nil {
		return
	}
	namespace := ""
	switch spec.Type {
	case reader.NodeSymbol:
		namespace = spec.Value
	case reader.NodeVector, reader.NodeList:
		if len(spec.Children) == 0 || spec.Children[0] == nil || spec.Children[0].Type != reader.NodeSymbol {
			c.ambiguous = true
			return
		}
		namespace = spec.Children[0].Value
		for index := 1; index < len(spec.Children); index++ {
			child := spec.Children[index]
			if child == nil {
				continue
			}
			if child.Type == reader.NodeSymbol || (child.Type == reader.NodeVector && !referAll) {
				// Prefix list such as [clojush random util]: the referred
				// namespaces are not individually known here.
				c.ambiguous = true
				return
			}
			if !referAll && child.Type == reader.NodeKeyword && child.Value == ":only" && index+1 < len(spec.Children) {
				for _, name := range spec.Children[index+1].Children {
					if name != nil && name.Type == reader.NodeSymbol {
						c.excluded[name.Value] = true
					}
				}
				return
			}
		}
	default:
		c.ambiguous = true
		return
	}
	collisions, known := knownNamespaceCoreCollisions[namespace]
	if !known {
		c.ambiguous = true
		return
	}
	for _, name := range collisions {
		c.excluded[name] = true
	}
}

func (c *coreReferral) readReferClojure(options []*reader.RichNode) {
	for index := 0; index+1 < len(options); index += 2 {
		option, value := options[index], options[index+1]
		if option == nil || value == nil {
			continue
		}
		if value.Type == reader.NodeQuote && len(value.Children) == 1 {
			value = value.Children[0]
		}
		names := map[string]bool{}
		for _, item := range value.Children {
			if item != nil && item.Type == reader.NodeSymbol {
				names[item.Value] = true
			}
		}
		switch option.Value {
		case ":exclude":
			for name := range names {
				c.excluded[name] = true
			}
		case ":only":
			c.only = names
		case ":rename":
			// Map entries alternate original and new names; the original is
			// no longer referred and the new name is not proven core either.
			for name := range names {
				c.excluded[name] = true
			}
		}
	}
}

func containsReferAll(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	for index, child := range node.Children {
		if child != nil && child.Type == reader.NodeKeyword && child.Value == ":refer" &&
			index+1 < len(node.Children) && node.Children[index+1] != nil &&
			node.Children[index+1].Type == reader.NodeKeyword && node.Children[index+1].Value == ":all" {
			return true
		}
		if containsReferAll(child) {
			return true
		}
	}
	return false
}

func rootCoreReferral(scope *Scope) *coreReferral {
	root := scope
	for root != nil && root.parent != nil {
		root = root.parent
	}
	if root == nil {
		return nil
	}
	return root.coreReferral
}

// coreIndependentProofRules prove their finding without relying on the
// identity of an unqualified clojure.core name.
var coreIndependentProofRules = map[string]bool{
	// The proof is a count of explicitly referred vars in the ns form.
	"excessive-refers": true,
}

// demoteWhenCoreNamesAmbiguous keeps the finding but withdraws a proof that
// depends on unqualified clojure.core names when the file refers a whole
// namespace: Clojure replaces a core mapping with a same-named var of that
// namespace (only warning), so the name's meaning is external evidence.
func demoteWhenCoreNamesAmbiguous(finding *rules.Finding, scope *Scope) {
	if finding == nil || finding.Confidence != rules.ConfidenceProven || finding.Contextual ||
		coreIndependentProofRules[finding.RuleID] || !scope.CoreNamesAmbiguous() {
		return
	}
	rules.SetContextualFindingWithEvidence(finding,
		"The namespace refers every var of another namespace (:use, :refer :all, or use/refer), which can replace clojure.core names; the core identity of the operators is not proven.",
		"referred-namespace-contents")
}
