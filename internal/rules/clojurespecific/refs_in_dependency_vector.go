package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type RefsInDependencyVectorRule struct{ rules.Rule }

func (r *RefsInDependencyVectorRule) Meta() rules.Rule { return r.Rule }

func dependencyNameLooksMutable(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{"atom", "ratom", "cursor", "-ref", "_ref", "state"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func dependencyIsLocallyBoundAtom(name string, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil {
			continue
		}
		bindings := ancestor
		if coreAsyncOperation(ancestor) == "let" && len(ancestor.Children) > 1 {
			bindings = ancestor.Children[1]
		}
		if bindings.Type != reader.NodeVector {
			continue
		}
		for index := 0; index+1 < len(bindings.Children); index += 2 {
			binding, value := bindings.Children[index], bindings.Children[index+1]
			if binding.Type == reader.NodeSymbol && binding.Value == name &&
				coreAsyncHasSuffix(coreAsyncOperation(value), "atom") {
				return true
			}
		}
	}
	return false
}

func (r *RefsInDependencyVectorRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	op := coreAsyncOperation(node)
	if !coreAsyncHasSuffix(op, "use-effect", "useEffect") || len(node.Children) < 3 {
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
		if dependencyNameLooksMutable(dependency.Value) || dependencyIsLocallyBoundAtom(dependency.Value, context) {
			return &rules.Finding{
				RuleID: r.ID, Filepath: filepath, Location: dependency.Location, Severity: r.Severity,
				Message: fmt.Sprintf("Mutable reference %q is used directly in an effect dependency vector; depend on its dereferenced value instead.", dependency.Value),
			}
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
