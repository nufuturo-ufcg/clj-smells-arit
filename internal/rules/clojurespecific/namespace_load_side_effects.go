package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

// NamespaceLoadSideEffectsRule detects require/use/import in problematic locations:
// 1. Inside function bodies (defn, fn)
// 2. At top level AFTER other definitions (after ns, defn, def, etc.)
//
// The idiomatic Clojure pattern is to declare all dependencies in the (ns ...) :require block.
// A top-level require immediately after (ns ...) is tolerated (script style).
// The actual smell is when require appears AFTER definitions in the file.
type NamespaceLoadSideEffectsRule struct {
	rules.Rule
}

func (r *NamespaceLoadSideEffectsRule) Meta() rules.Rule {
	return r.Rule
}

func isLoadTimeSideEffectCall(node *reader.RichNode) bool {
	return rules.CallResolvesTo(node,
		"clojure.core/require", "clojure.core/use", "clojure.core/import", "clojure.core/load-file")
}

func isLazyLoadCall(node *reader.RichNode) bool {
	return rules.CallResolvesTo(node, "clojure.core/requiring-resolve")
}

func hasProvenLocalNamespaceLoadEffect(node *reader.RichNode, context map[string]interface{}) bool {
	summaries := rules.FunctionSummaries(context)
	if len(summaries) == 0 || node == nil || node.Type != reader.NodeList {
		return false
	}
	candidates := []string{semantics.CallName(node)}
	if len(node.Children) > 0 && node.Children[0] != nil {
		candidates = append(candidates, node.Children[0].Value)
	}
	for _, candidate := range candidates {
		if summary, found := summaries[candidate]; found && summary.Evidence == semantics.EvidenceProven &&
			summary.Effects.Has(semantics.EffectNamespaceLoad) {
			return true
		}
	}
	return false
}

func (r *NamespaceLoadSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	lowerPath := strings.ToLower(filepath)
	if strings.HasSuffix(lowerPath, "project.clj") || strings.HasSuffix(lowerPath, "deps.edn") {
		return nil
	}

	if node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol {
		return nil
	}

	symbol := node.Children[0].Value

	if symbol == "ns" {
		return nil
	}
	if !rules.ExecutesAtLoad(context) {
		return nil
	}

	isLoadTime := isLoadTimeSideEffectCall(node)
	isLazyLoad := isLazyLoadCall(node)
	isLocalLoadTime := hasProvenLocalNamespaceLoadEffect(node, context)

	if !isLoadTime && !isLazyLoad && !isLocalLoadTime {
		return nil
	}

	isInsideNs := false
	isInsideDefn := false

	if enclosing, ok := context["enclosingForms"].([]string); ok {
		for _, f := range enclosing {
			if f == "ns" {
				isInsideNs = true
			}
			if f == "defn" || f == "defn-" || f == "fn" || f == "letfn" || f == "__fn-literal__" {
				isInsideDefn = true
			}
		}
	}

	// Inside (ns ...), this is the correct form
	if isInsideNs {
		return nil
	}

	// requiring-resolve inside a function is deliberate lazy loading and is acceptable
	if isLazyLoad && isInsideDefn {
		return nil
	}

	// `(require ns-sym)` is a deliberate runtime plugin-loading pattern. Static
	// dependency rules cannot prove it should be eager, so prefer a false
	// negative to a false positive.
	if symbol == "require" && len(node.Children) == 2 && node.Children[1].Type == reader.NodeSymbol {
		return nil
	}
	// Smell 1: require/use/import INSIDE a function body
	if isInsideDefn {
		severity := rules.ContextualSeverity(context, r.Severity)
		tags := rules.ContextualTags(context)
		if hasOptionalLoadGuard(context) {
			severity = rules.SeverityHint
			tags = append(tags, "conditional-load")
		}
		finding := &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Side effect: '%s' called inside a function body. Move namespace dependencies to the (ns ...) :require form.", symbol),
			Filepath: filepath,
			Location: node.Location,
			Severity: severity,
			Tags:     tags,
		}
		return rules.SetContextualFindingWithEvidence(finding, "Loading inside a function or guard may be deliberate for plugins, compatibility, or optional loading.", "plugin-lifecycle", "optional-load-contract")
	}

	// A direct top-level form is tolerated for scripts. Nested top-level
	// execution (def initializers, conditionals, try, let, etc.) is hidden
	// load-time behavior and is therefore reportable.
	if parent, ok := context["parent"].(*reader.RichNode); ok && parent != nil {
		severity := rules.ContextualSeverity(context, r.Severity)
		tags := rules.ContextualTags(context)
		if hasOptionalLoadGuard(context) {
			severity = rules.SeverityHint
			tags = append(tags, "conditional-load")
		}
		finding := &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"Namespace load side effect: '%s' is nested in a load-time expression. "+
					"All namespace dependencies should be declared inside the (ns ...) macro at the top of the file.",
				symbol,
			),
			Filepath: filepath,
			Location: node.Location,
			Severity: severity,
			Tags:     tags,
		}
		if hasOptionalLoadGuard(context) {
			return rules.SetContextualFindingWithEvidence(finding, "Conditional loading may be deliberate for compatibility or optional availability.", "compatibility-contract", "optional-dependency")
		}
		return finding
	}

	// A direct load form after an executable top-level form is still evaluated
	// during namespace loading. Keep it contextual because scripts and plugin
	// loaders may intentionally use this ordering.
	if afterExecutable, _ := context["top-level-after-executable"].(bool); afterExecutable {
		finding := &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Namespace load side effect: '%s' appears after executable top-level forms. Prefer declaring the dependency in the (ns ...) :require form.", symbol),
			Filepath: filepath,
			Location: node.Location,
			Severity: rules.ContextualSeverity(context, r.Severity),
			Tags:     rules.ContextualTags(context),
		}
			return rules.SetContextualFindingWithEvidence(finding, "Loading after executable code may be deliberate in scripts, plugins, or initializers.", "top-level-load-order", "plugin-lifecycle")
	}

	return nil
}

func hasOptionalLoadGuard(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		head := strings.TrimPrefix(ancestor.Children[0].Value, "clojure.core/")
		switch head {
		case "if", "when", "when-not", "if-not":
			if len(ancestor.Children) > 1 && containsOptionalGuardCall(ancestor.Children[1]) {
				return true
			}
		case "try":
			for _, child := range ancestor.Children[1:] {
				if child != nil && child.Type == reader.NodeList && len(child.Children) > 0 && child.Children[0].Type == reader.NodeSymbol && child.Children[0].Value == "catch" {
					return true
				}
			}
		}
	}
	return false
}

func containsOptionalGuardCall(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		head := node.Children[0].Value
		if head == "resolve" || head == "clojure.core/resolve" ||
			head == "System/getProperty" || head == "System/getenv" ||
			head == "java.lang.System/getProperty" || head == "java.lang.System/getenv" {
			return true
		}
	}
	for _, child := range node.Children {
		if containsOptionalGuardCall(child) {
			return true
		}
	}
	return false
}

func init() {
	rules.RegisterRule(&NamespaceLoadSideEffectsRule{
		Rule: rules.Rule{
			ID:          "namespace-load-side-effects",
			Name:        "Namespace Load Side Effects",
			Description: "Using require, use, or import inside function bodies or after other top-level definitions introduces hidden dependencies. Declare all namespace dependencies in the (ns ...) :require form.",
			Severity:    rules.SeverityWarning,
		},
	})
}
