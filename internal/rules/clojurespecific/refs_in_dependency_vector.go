package clojurespecific

import (
	"fmt"

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
	if resolved == nil || resolved.Kind == reader.ResolutionLocal || resolved.Kind == reader.ResolutionUnresolved {
		return false
	}
	switch resolved.CanonicalName {
	case "clojure.core/atom", "clojure.core/ref", "reagent.core/atom":
		return true
	default:
		return false
	}
}

func isResolvedEffectHook(node *reader.RichNode) bool {
	resolved := rules.ResolvedCall(node)
	if resolved == nil || resolved.Kind == reader.ResolutionLocal || resolved.Kind == reader.ResolutionUnresolved {
		return false
	}
	switch resolved.CanonicalName {
	case "reagent.core/use-effect", "reagent.core/useEffect":
		return true
	default:
		return false
	}
}

func (r *RefsInDependencyVectorRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if !isResolvedEffectHook(node) || len(node.Children) < 3 {
		return nil
	}
	dependencies := node.Children[len(node.Children)-1]
	if dependencies.Type != reader.NodeVector {
		return nil
	}
	for _, dependency := range dependencies.Children {
		if dependency.Type != reader.NodeSymbol {
			continue
		}
		if dependencyIsLocallyBoundAtom(dependency.Value, context) {
			return rules.SetContextualFindingWithEvidence(&rules.Finding{
				RuleID: r.ID, Filepath: filepath, Location: dependency.Location, Severity: r.Severity,
				Message: fmt.Sprintf("Mutable reference %q is used directly in an effect dependency vector; depend on its dereferenced value instead.", dependency.Value),
			}, "The ref identity may be a deliberate effect dependency; the AST does not prove that the dereferenced value is the correct contract.", "effect-dependency-contract", "dereferenced-value-contract")
		}
	}
	return nil
}

func init() {
	rules.RegisterRule(&RefsInDependencyVectorRule{Rule: rules.Rule{
		ID: "refs-in-dependency-vector", Name: "Refs in Dependency Vector",
		Description: "Detects statically identifiable atom, ratom, ref, cursor, or state objects passed directly to use-effect dependencies.",
		Severity:    rules.SeverityWarning,
	}})
}
