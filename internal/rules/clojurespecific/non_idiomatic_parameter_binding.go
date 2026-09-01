package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

// nonIdiomaticParameterBindingRule detects the & [x] pattern for capturing
// a single optional function parameter. This pattern is non-idiomatic in Clojure:
// it forces callers to handle variadic arity when only one optional parameter is needed.
// The idiomatic form is to use multiple function arities or an options map.
//
// Detects:
//
//	(defn f [x & [y]] ...)       → smell: & [y] with only one element in the rest vector
//	(defn f [x & [y z]] ...)     → smell: & [y z] — should use an options map
//	(fn [x & [y]] ...)           → smell: lambdas as well
//	(defn- f [x & [y]] ...)      → smell: private functions as well
//
// Does not detect:
//
//	(defn f [x & args] ...)      → valid: open variadic capture
//	(defmacro m [& body] ...)    → valid: macros with variadic bodies
type nonIdiomaticParameterBindingRule struct {
	rules.Rule
}

func (r *nonIdiomaticParameterBindingRule) Meta() rules.Rule {
	return r.Rule
}

// findParamVectors returns every parameter vector belonging to a defn/fn/defn-.
// The structure is either (defn name docstring? [params] body) or a sequence
// of arity clauses: (defn name docstring? ([params] body) ...). Looking at all
// clauses matters because a smell can be present only in one overload.
func findParamVectors(node *reader.RichNode) []*reader.RichNode {
	if node == nil || len(node.Children) < 2 {
		return nil
	}

	start := 1
	if node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value != "fn" {
		start = 2 // skip the defn/defn- name
	} else if node.Children[0].Type == reader.NodeSymbol && node.Children[0].Value == "fn" &&
		start < len(node.Children) && node.Children[start].Type == reader.NodeSymbol {
		start++ // skip an optional name on a named fn
	}
	for start < len(node.Children) {
		child := node.Children[start]
		if child.Type == reader.NodeString || child.Type == reader.NodeMap {
			start++ // optional docstring or metadata
			continue
		}
		break
	}

	if start >= len(node.Children) {
		return nil
	}
	if node.Children[start].Type == reader.NodeVector {
		return []*reader.RichNode{node.Children[start]}
	}

	var vectors []*reader.RichNode
	for _, child := range node.Children[start:] {
		if child.Type != reader.NodeList || len(child.Children) == 0 || child.Children[0].Type != reader.NodeVector {
			break
		}
		vectors = append(vectors, child.Children[0])
	}
	return vectors
}

// hasRestDestructuring checks whether a parameter vector has & [x ...] (rest destructuring)
// and returns (true, element count, description) if found.
func hasRestDestructuring(paramVec *reader.RichNode) (bool, int, string) {
	if paramVec == nil || paramVec.Type != reader.NodeVector {
		return false, 0, ""
	}
	children := paramVec.Children
	for i := 0; i < len(children)-1; i++ {
		// Look for the "&" symbol
		if children[i].Type == reader.NodeSymbol && children[i].Value == "&" {
			next := children[i+1]
			// The problematic pattern: & [x] or & [x y z] — destructured rest vector
			if next.Type == reader.NodeVector {
				elemCount := 0
				names := []string{}
				for _, el := range next.Children {
					if el.Type == reader.NodeSymbol && el.Value != "&" {
						elemCount++
						names = append(names, el.Value)
					}
				}
				if elemCount > 0 {
					descr := "["
					for j, n := range names {
						if j > 0 {
							descr += " "
						}
						descr += n
					}
					descr += "]"
					return true, elemCount, descr
				}
			}
		}
	}
	return false, 0, ""
}

func (r *nonIdiomaticParameterBindingRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}
	if r.IsInside(context, "__non-evaluated__", "comment") ||
		rules.CurrentExecutionContext(context) == rules.ExecutionUnknown ||
		parameterBindingIsInsideMacroDefinition(context) {
		return nil
	}

	first := node.Children[0]
	if first.Type != reader.NodeSymbol {
		return nil
	}

	sym := first.Value

	// Check only defn, defn-, and fn (not defmacro — variadic rest args in macros are idiomatic)
	if sym != "defn" && sym != "defn-" && sym != "fn" {
		return nil
	}
	if first.Resolution != nil {
		switch first.Resolution.Kind {
		case reader.ResolutionLocal, reader.ResolutionNamespaceVar, reader.ResolutionJavaStatic,
			reader.ResolutionJavaConstructor, reader.ResolutionJavaMethod:
			return nil
		case reader.ResolutionClojureCore:
			if sym == "fn" && first.Resolution.CanonicalName != "clojure.core/fn" {
				return nil
			}
		}
	}
	if sym != "fn" && parameterBindingIsInsideLexicalBinding(context) {
		return nil
	}

	// Inspect every arity. A multi-arity function may expose the problematic
	// destructuring in only one of its clauses.
	var elemCount int
	var restDescr string
	for _, paramVec := range findParamVectors(node) {
		hasRest, count, descr := hasRestDestructuring(paramVec)
		if hasRest {
			elemCount, restDescr = count, descr
			break
		}
	}
	if restDescr == "" {
		return nil
	}

	// Determine the refactoring message based on the number of parameters
	var suggestion string
	if elemCount == 1 {
		suggestion = "use multiple arities: ([x] (f x default)) ([x opt] ...)"
	} else {
		suggestion = "use an options map: ([x {:keys [" + restDescr[1:len(restDescr)-1] + "] :or {...}}] ...)"
	}

	fnName := ""
	if sym != "fn" && len(node.Children) >= 2 && node.Children[1].Type == reader.NodeSymbol {
		fnName = " '" + node.Children[1].Value + "'"
	}

	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Non-idiomatic parameter binding: function%s uses `& %s` for optional params. "+
				"This hides the real arity and confuses callers. Instead, %s.",
			fnName, restDescr, suggestion,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}, "The parameter shape may be the deliberate contract of an optional API; the AST does not prove that multiple arities or a map would be equivalent.", "public-arity-contract", "optional-parameter-contract")
}

func parameterBindingIsInsideMacroDefinition(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		if ancestor.Children[0].Value == "defmacro" {
			return true
		}
	}
	return false
}

func parameterBindingIsInsideLexicalBinding(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		switch ancestor.Children[0].Value {
		case "let", "let*", "loop", "loop*", "binding", "with-open":
			return true
		}
	}
	return false
}

func init() {
	rules.RegisterRule(&nonIdiomaticParameterBindingRule{
		Rule: rules.Rule{
			ID:          "non-idiomatic-parameter-binding",
			Name:        "Non-Idiomatic Parameter Binding",
			Description: "Detects functions using & [x] destructuring for optional parameters. This pattern hides the real arity and is non-idiomatic. Prefer multiple arities or an options map.",
			Severity:    rules.SeverityInfo,
		},
	})
}
