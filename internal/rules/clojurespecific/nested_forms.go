package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

// NestedFormsRule intentionally keeps the legacy configuration fields so old
// configuration files continue to load. MaxConditionalDepth and TrackedForms
// are deprecated: depth and membership in a broad form list are not evidence
// that a semantics-preserving rewrite exists.
type NestedFormsRule struct {
	rules.Rule
	MaxConsecutiveSameForms int      `json:"max_consecutive_same_forms" yaml:"max_consecutive_same_forms"`
	MaxConditionalDepth     int      `json:"max_conditional_depth" yaml:"max_conditional_depth"`
	TrackedForms            []string `json:"tracked_forms" yaml:"tracked_forms"`
}

type nestedFormsCandidate struct {
	subtype string
	forms   []*reader.RichNode
}

func (r *NestedFormsRule) Meta() rules.Rule {
	return r.Rule
}

func (r *NestedFormsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || r.isExcludedContext(context) || r.isContinuationOfParentCandidate(node, context) {
		return nil
	}

	candidate := r.safeCandidate(node)
	if candidate == nil {
		return nil
	}

	var message string
	switch candidate.subtype {
	case "nested-let-flattening":
		message = fmt.Sprintf(
			"Safe nested let flattening detected (%d consecutive forms). The direct bodies contain no additional expressions and the binding vectors can be concatenated in evaluation order.",
			len(candidate.forms),
		)
	case "nested-doseq-flattening":
		message = fmt.Sprintf(
			"Safe nested doseq flattening detected (%d consecutive forms). The inner doseq is the sole body expression and its bindings can be appended in iteration order.",
			len(candidate.forms),
		)
	case "nested-when-flattening":
		message = fmt.Sprintf(
			"Safe nested when flattening detected (%d consecutive forms). The inner when is the sole body expression and conditions can be combined with and.",
			len(candidate.forms),
		)
	default:
		return nil
	}

	return &rules.Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func (r *NestedFormsRule) isExcludedContext(context map[string]interface{}) bool {
	if r.IsInside(context, "__non-evaluated__", "delay", "lazy-seq", "future", "future-call") {
		return true
	}

	// A candidate rooted in a binding vector is an initializer, not a chain in
	// the form's direct body. The inner nodes are then suppressed as
	// continuations of that excluded root.
	parent, _ := context["parent"].(*reader.RichNode)
	return parent != nil && parent.Type == reader.NodeVector
}

func (r *NestedFormsRule) safeCandidate(node *reader.RichNode) *nestedFormsCandidate {
	switch {
	case resolvesToCoreForm(node, "let"):
		forms, ok := collectDirectChain(node, "let")
		if !ok || len(forms) < r.letThreshold() || !bindingsRemainDistinct(forms, false) {
			return nil
		}
		return &nestedFormsCandidate{subtype: "nested-let-flattening", forms: forms}

	case resolvesToCoreForm(node, "doseq"):
		forms, ok := collectDirectChain(node, "doseq")
		if !ok || len(forms) < 2 || !bindingsRemainDistinct(forms, true) {
			return nil
		}
		return &nestedFormsCandidate{subtype: "nested-doseq-flattening", forms: forms}

	case resolvesToCoreForm(node, "when"):
		forms, ok := collectDirectChain(node, "when")
		if !ok || len(forms) < 3 {
			return nil
		}
		return &nestedFormsCandidate{subtype: "nested-when-flattening", forms: forms}
	}

	return nil
}

func (r *NestedFormsRule) letThreshold() int {
	if r.MaxConsecutiveSameForms < 2 {
		return 2
	}
	return r.MaxConsecutiveSameForms
}

// collectDirectChain follows only the sole body expression. It never searches
// binding initializers, branches, do blocks, exception forms, loops, functions,
// or unknown macros.
func collectDirectChain(root *reader.RichNode, formName string) ([]*reader.RichNode, bool) {
	forms := []*reader.RichNode{}
	current := root
	for resolvesToCoreForm(current, formName) {
		if formName == "when" {
			if current == nil || len(current.Children) != 3 {
				return nil, false
			}
		} else if !validBindingForm(current) {
			return nil, false
		}
		forms = append(forms, current)

		body, ok := soleBodyExpression(current)
		if !ok || !resolvesToCoreForm(body, formName) {
			break
		}
		current = body
	}
	return forms, true
}

func validBindingForm(node *reader.RichNode) bool {
	return node != nil && len(node.Children) == 3 &&
		node.Children[1] != nil && node.Children[1].Type == reader.NodeVector
}

func soleBodyExpression(node *reader.RichNode) (*reader.RichNode, bool) {
	if node == nil || len(node.Children) != 3 {
		return nil, false
	}
	return node.Children[2], node.Children[2] != nil
}

func resolvesToCoreForm(node *reader.RichNode, formName string) bool {
	if rules.CallResolvesTo(node, "clojure.core/"+formName) {
		return true
	}
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	// The analyzer's core symbol catalog does not yet enumerate every clojure.core
	// macro. An unresolved, exact unqualified name is safe here; a shadowing local
	// definition is explicitly classified as ResolutionLocal and rejected.
	return head != nil && head.Type == reader.NodeSymbol && head.Value == formName &&
		head.Resolution != nil && head.Resolution.Kind == reader.ResolutionUnresolved
}

func (r *NestedFormsRule) isContinuationOfParentCandidate(node *reader.RichNode, context map[string]interface{}) bool {
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil {
		return false
	}
	parentBody, ok := soleBodyExpression(parent)
	if !ok || parentBody != node {
		return false
	}
	return (resolvesToCoreForm(parent, "let") && resolvesToCoreForm(node, "let")) ||
		(resolvesToCoreForm(parent, "doseq") && resolvesToCoreForm(node, "doseq")) ||
		(resolvesToCoreForm(parent, "when") && resolvesToCoreForm(node, "when"))
}

// bindingsRemainDistinct rejects flattening that would turn legal lexical
// shadowing in nested forms into duplicate names in one binding vector. For
// doseq it also validates the modifier grammar conservatively.
func bindingsRemainDistinct(forms []*reader.RichNode, doseqBindings bool) bool {
	seen := make(map[string]struct{})
	for _, form := range forms {
		bindings := form.Children[1]
		var names []string
		var ok bool
		if doseqBindings {
			names, ok = namesBoundByDoseq(bindings)
		} else {
			names, ok = namesBoundByLet(bindings)
		}
		if !ok {
			return false
		}
		for _, name := range names {
			if name == "_" || name == "&" {
				continue
			}
			if _, duplicate := seen[name]; duplicate {
				return false
			}
			seen[name] = struct{}{}
		}
	}
	return true
}

func namesBoundByLet(bindings *reader.RichNode) ([]string, bool) {
	if bindings == nil || bindings.Type != reader.NodeVector {
		return nil, false
	}
	names := []string{}
	for i := 0; i < len(bindings.Children); {
		if i+1 >= len(bindings.Children) || bindings.Children[i] == nil || bindings.Children[i+1] == nil {
			return nil, false
		}

		bindingIndex := i
		valueIndex := i + 1
		if i+2 < len(bindings.Children) && looksLikeTypeTag(bindings.Children[i]) &&
			bindings.Children[i+1].Type == reader.NodeSymbol {
			bindingIndex = i + 1
			valueIndex = i + 2
		}
		if bindings.Children[valueIndex] == nil {
			return nil, false
		}
		collectBindingNames(bindings.Children[bindingIndex], &names)
		i = valueIndex + 1
	}
	return names, true
}

func looksLikeTypeTag(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeSymbol || node.Value == "" {
		return false
	}
	if strings.Contains(node.Value, ".") || (node.Value[0] >= 'A' && node.Value[0] <= 'Z') {
		return true
	}
	switch node.Value {
	case "boolean", "byte", "char", "short", "int", "long", "float", "double", "objects", "bytes", "chars", "shorts", "ints", "longs", "floats", "doubles", "booleans":
		return true
	default:
		return false
	}
}

func namesBoundByDoseq(bindings *reader.RichNode) ([]string, bool) {
	if bindings == nil || bindings.Type != reader.NodeVector {
		return nil, false
	}
	names := []string{}
	for i := 0; i < len(bindings.Children); {
		child := bindings.Children[i]
		if child == nil || i+1 >= len(bindings.Children) || bindings.Children[i+1] == nil {
			return nil, false
		}

		if child.Type == reader.NodeKeyword {
			switch child.Value {
			case ":let":
				letBindings := bindings.Children[i+1]
				letNames, ok := namesBoundByLet(letBindings)
				if !ok {
					return nil, false
				}
				names = append(names, letNames...)
			case ":when", ":while":
			default:
				return nil, false
			}
			i += 2
			continue
		}

		collectBindingNames(child, &names)
		i += 2
	}
	return names, true
}

func collectBindingNames(node *reader.RichNode, names *[]string) {
	if node == nil {
		return
	}
	if node.Type == reader.NodeSymbol {
		*names = append(*names, node.Value)
		return
	}
	for _, child := range node.Children {
		collectBindingNames(child, names)
	}
}

func init() {
	defaultRule := &NestedFormsRule{
		Rule: rules.Rule{
			ID:          "nested-forms",
			Name:        "Nested Forms",
			Description: "Detects directly nested binding forms only when their binding vectors can be flattened without changing evaluation order, scope, or body execution.",
			Severity:    rules.SeverityWarning,
		},
		MaxConsecutiveSameForms: 3,
		MaxConditionalDepth:     4,
		TrackedForms: []string{
			"let", "when", "if", "when-let", "if-let", "when-some", "if-some",
			"when-not", "if-not", "loop", "binding", "with-open", "with-local-vars",
			"doseq", "dotimes", "for", "try", "catch", "cond", "case",
		},
	}

	rules.RegisterRule(defaultRule)
}
