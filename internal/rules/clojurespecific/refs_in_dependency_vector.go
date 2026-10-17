package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type RefsInDependencyVectorRule struct{ rules.Rule }

func (r *RefsInDependencyVectorRule) Meta() rules.Rule { return r.Rule }

func dependencyIsLocallyBoundAtom(name string, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for ancestorIndex := len(ancestors) - 1; ancestorIndex >= 0; ancestorIndex-- {
		ancestor := ancestors[ancestorIndex]
		if ancestor == nil {
			continue
		}
		if !isReferenceBindingForm(ancestor) || len(ancestor.Children) <= 1 {
			continue
		}
		bindings := ancestor.Children[1]
		if bindings.Type != reader.NodeVector {
			continue
		}
		for index := 0; index+1 < len(bindings.Children); index += 2 {
			binding, value := bindings.Children[index], bindings.Children[index+1]
			if binding.Type == reader.NodeSymbol && binding.Value == name &&
				isKnownMutableReferenceConstructor(value) {
				return true
			}
			if binding.Type == reader.NodeSymbol && binding.Value == name {
				return false
			}
		}
	}
	return false
}

// These forms introduce a local binding whose initializer is evaluated before
// the effect body. The raw-name fallback is intentional for let*/if-let/
// when-let: the analyzer may leave these special forms unresolved while the
// binding shape itself remains unambiguous. loop is excluded because recur can
// replace the initial reference with a value of a different contract.
func isReferenceBindingForm(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return false
	}
	if head.Resolution != nil && head.Resolution.Kind != reader.ResolutionUnresolved &&
		head.Resolution.Kind != reader.ResolutionLocal {
		switch head.Resolution.CanonicalName {
		case "clojure.core/let", "clojure.core/let*", "clojure.core/if-let", "clojure.core/when-let":
			return true
		default:
			return false
		}
	}
	switch head.Value {
	case "let", "let*", "if-let", "when-let":
		return true
	default:
		return false
	}
}

func isKnownMutableReferenceConstructor(node *reader.RichNode) bool {
	resolved := rules.ResolvedCall(node)
	if resolved == nil {
		return false
	}
	if resolved.Kind != reader.ResolutionLocal && resolved.Kind != reader.ResolutionUnresolved {
		switch resolved.CanonicalName {
		case "clojure.core/atom", "clojure.core/ref", "reagent.core/atom", "reagent.core/cursor",
			"reagent.core/track", "reagent.ratom/atom", "reagent.ratom/make-reaction",
			"uix.core/use-ref", "helix.hooks/use-ref", "rum.core/create-ref":
			return true
		}
	}
	// String requires such as ["react" :as react] are not registered as
	// aliases, so React refs are recognized by the member name.
	switch hookMemberName(node) {
	case "useRef", "createRef":
		return true
	}
	return false
}

func hookMemberName(node *reader.RichNode) string {
	if node == nil || (node.Type != reader.NodeList && node.Type != reader.NodeFnLiteral) || len(node.Children) == 0 {
		return ""
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return ""
	}
	name := head.Value
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		name = name[slash+1:]
	}
	return name
}

// effectHookDependencies returns the dependency collection of a recognized
// effect/memo hook call, or nil. Callback hooks (use-callback/useCallback) are
// excluded: a callback dereferences the reference when invoked, so depending
// on the reference identity is correct there. Known libraries are matched by canonical
// name; project wrappers (hooks/use-effect!) and string-required React
// (react/useEffect) are matched by the member name.
func effectHookDependencies(node *reader.RichNode) *reader.RichNode {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}
	if resolved := rules.ResolvedCall(node); resolved != nil && strings.HasPrefix(resolved.CanonicalName, "helix.hooks/") {
		// helix takes the dependency vector as the first argument.
		switch strings.TrimPrefix(resolved.CanonicalName, "helix.hooks/") {
		case "use-effect", "use-layout-effect", "use-memo":
			return dependencyCollection(node.Children[1], nil)
		}
		return nil
	}
	resolved := rules.ResolvedCall(node)
	known := false
	if resolved != nil && resolved.Kind != reader.ResolutionLocal && resolved.Kind != reader.ResolutionUnresolved {
		switch resolved.CanonicalName {
		case "reagent.core/use-effect", "reagent.core/useEffect",
			"uix.core/use-effect", "uix.core/use-layout-effect", "uix.core/use-memo":
			known = true
		}
	}
	if !known {
		// A qualified kebab-case name alone does not prove the hook contract
		// (example/use-effect). React's camelCase hooks are accepted by name
		// because string requires (["react" :as react]) have no alias entry.
		switch hookMemberName(node) {
		case "useEffect", "useLayoutEffect", "useMemo":
			if resolved == nil || resolved.Kind == reader.ResolutionUnresolved {
				known = true
			}
		}
	}
	if !known {
		return nil
	}
	last := node.Children[len(node.Children)-1]
	var previous *reader.RichNode
	if len(node.Children) >= 2 {
		previous = node.Children[len(node.Children)-2]
	}
	return dependencyCollection(last, previous)
}

// dependencyCollection accepts [deps] and #js [deps] (as a tagged node or as
// a tag followed by the vector).
func dependencyCollection(node, previous *reader.RichNode) *reader.RichNode {
	if node == nil {
		return nil
	}
	if node.Type == reader.NodeVector {
		return node
	}
	if node.Type == reader.NodeTag && len(node.Children) == 1 && node.Children[0] != nil && node.Children[0].Type == reader.NodeVector {
		return node.Children[0]
	}
	_ = previous
	return nil
}

// dependencyUsedAsReference reports whether the hook body treats the symbol
// as a reference: @x, (deref x), (.-current x), (reset! x ...), (swap! x ...).
func dependencyUsedAsReference(name string, node *reader.RichNode, deps *reader.RichNode) bool {
	if node == nil || node == deps {
		return false
	}
	switch node.Type {
	case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeReaderDiscard:
		return false
	case reader.NodeDeref:
		if len(node.Children) == 1 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == name {
			return true
		}
	case reader.NodeList:
		if len(node.Children) >= 2 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
			target := node.Children[1]
			head := strings.TrimPrefix(node.Children[0].Value, "clojure.core/")
			switch head {
			case "deref", "reset!", "swap!", ".-current", "reset-vals!", "swap-vals!":
				if target != nil && target.Type == reader.NodeSymbol && target.Value == name {
					return true
				}
			}
		}
	}
	for _, child := range node.Children {
		if dependencyUsedAsReference(name, child, deps) {
			return true
		}
	}
	return false
}

// nearestLetBinding reports whether the innermost binding of name is a
// let-like binding (not loop: recur can replace the initial reference; not a
// function parameter: a reference parameter may be a deliberate contract).
func nearestLetBinding(name string, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for i := len(ancestors) - 1; i >= 0; i-- {
		form := ancestors[i]
		if form == nil || form.Type != reader.NodeList || len(form.Children) < 2 ||
			form.Children[0] == nil || form.Children[0].Type != reader.NodeSymbol || form.Children[1] == nil ||
			form.Children[1].Type != reader.NodeVector {
			continue
		}
		head := strings.TrimPrefix(form.Children[0].Value, "clojure.core/")
		switch head {
		case "let", "let*", "loop", "loop*", "if-let", "when-let", "when-some", "if-some", "binding", "with-open":
		default:
			continue
		}
		bindings := form.Children[1].Children
		for k := 0; k+1 < len(bindings); k += 2 {
			if bindings[k] != nil && bindings[k].Type == reader.NodeSymbol && bindings[k].Value == name {
				return head != "loop" && head != "loop*"
			}
		}
	}
	return false
}

// dependencyBoundAtTopLevel reports a def/defonce whose value is a reference
// constructor, e.g. (defonce app-state (r/atom {})).
func dependencyBoundAtTopLevel(dependency *reader.RichNode) bool {
	if dependency == nil || dependency.ResolvedDefinition == nil {
		return false
	}
	definition := dependency.ResolvedDefinition
	if definition.Type != reader.NodeList || len(definition.Children) < 3 || definition.Children[0] == nil {
		return false
	}
	switch definition.Children[0].Value {
	case "def", "defonce":
		return isKnownMutableReferenceConstructor(definition.Children[len(definition.Children)-1])
	}
	return false
}

func (r *RefsInDependencyVectorRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	dependencies := effectHookDependencies(node)
	if dependencies == nil {
		return nil
	}
	for _, dependency := range dependencies.Children {
		if dependency == nil || dependency.Type != reader.NodeSymbol {
			continue
		}
		evidence := ""
		switch {
		case dependencyIsLocallyBoundAtom(dependency.Value, context):
			evidence = "bound locally to a reference constructor"
		case dependencyBoundAtTopLevel(dependency):
			evidence = "defined at top level with a reference constructor"
		case nearestLetBinding(dependency.Value, context) && dependencyUsedAsReference(dependency.Value, node, dependencies):
			evidence = "dereferenced or updated as a reference inside the hook"
		}
		if evidence != "" {
			return rules.SetContextualFindingWithEvidence(&rules.Finding{
				RuleID: r.ID, Filepath: filepath, Location: dependency.Location, Severity: r.Severity,
				Message: fmt.Sprintf("Mutable reference %q is used directly in an effect dependency vector (%s); depend on its dereferenced value instead.", dependency.Value, evidence),
			}, "The ref identity may be a deliberate effect dependency; the AST does not prove that the dereferenced value is the correct contract.", "effect-dependency-contract", "dereferenced-value-contract")
		}
	}
	return nil
}

func init() {
	rules.RegisterRule(&RefsInDependencyVectorRule{Rule: rules.Rule{
		ID: "refs-in-dependency-vector", Name: "Refs in Dependency Vector",
		Description: "Detects atom, ratom, ref, cursor, or use-ref objects passed directly to effect/memo hook dependencies, using constructor bindings, top-level definitions, or reference use inside the hook as evidence.",
		Severity:    rules.SeverityWarning,
	}})
}
