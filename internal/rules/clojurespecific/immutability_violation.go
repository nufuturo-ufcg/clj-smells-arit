package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ImmutabilityViolationRule struct {
	rules.Rule
}

func (r *ImmutabilityViolationRule) Meta() rules.Rule {
	return r.Rule
}

func isLocalRuntimeScope(context map[string]interface{}) bool {
	insideFunction, _ := context["isInsideFunction"].(bool)
	return insideFunction
}

func isKnownCoreMutation(node *reader.RichNode, names ...string) bool {
	return rules.CallResolvesTo(node, names...)
}

func isJavaArrayTypeHint(typeHint string) bool {
	hint := strings.TrimSpace(typeHint)
	if strings.HasPrefix(hint, "[") || strings.HasSuffix(hint, "[]") {
		return true
	}
	// Only the plural array hints denote arrays; ^long or ^double hint a
	// scalar.
	switch hint {
	case "booleans", "bytes", "chars", "doubles", "floats", "ints", "longs", "objects", "shorts":
		return true
	default:
		return false
	}
}

// arrayOrigin classifies where a mutated array comes from. Only a caller's
// array (a parameter) or a Var is externally visible state; an array built in
// the same function is local construction, and any other origin (a field, an
// element of a collection, the result of an unknown call) is not proven.
type arrayOrigin int

const (
	arrayOriginUnknown arrayOrigin = iota
	arrayOriginLocalConstruction
	arrayOriginExternal
)

var localArrayConstructors = map[string]bool{
	"object-array": true, "make-array": true, "long-array": true, "int-array": true, "double-array": true,
	"float-array": true, "byte-array": true, "char-array": true, "short-array": true, "boolean-array": true,
	"aclone": true, "into-array": true, "to-array": true, "to-array-2d": true,
}

// constructsLocalArray accepts an array constructor call, or a reader
// conditional whose every branch builds a fresh array, e.g.
// #?(:clj (double-array 3) :cljs (js/Float32Array. 3)).
func constructsLocalArray(initializer *reader.RichNode) bool {
	if initializer == nil {
		return false
	}
	if initializer.Type == reader.NodeReaderCond {
		if len(initializer.Children) < 2 || len(initializer.Children)%2 != 0 {
			return false
		}
		for i := 1; i < len(initializer.Children); i += 2 {
			if !constructsLocalArray(initializer.Children[i]) {
				return false
			}
		}
		return true
	}
	if initializer.Type != reader.NodeList || len(initializer.Children) == 0 || initializer.Children[0] == nil {
		return false
	}
	head := initializer.Children[0].Value
	if localArrayConstructors[strings.TrimPrefix(head, "clojure.core/")] {
		return true
	}
	// JavaScript typed array constructors: (js/Float32Array. n)
	return strings.HasPrefix(head, "js/") && strings.HasSuffix(head, "Array.")
}

// isNamedFunctionParameter reports whether symbol resolves to a parameter of
// an enclosing defn, defn-, or defmethod: the caller's value. The resolved
// binding node is compared, not the name, so a let that shadows the parameter
// (e.g. (let [buf (byte-array 4)] ...)) is not mistaken for it. Anonymous fn
// parameters stay unproven, as for a direct mutation.
func isNamedFunctionParameter(symbol *reader.RichNode, context map[string]interface{}) bool {
	if symbol == nil || symbol.ResolvedDefinition == nil {
		return false
	}
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 {
			continue
		}
		head := ancestor.Children[0]
		if head == nil || head.Type != reader.NodeSymbol {
			continue
		}
		switch strings.TrimPrefix(head.Value, "clojure.core/") {
		case "defn", "defn-", "defmethod":
			if functionParamsContainNode(ancestor, symbol.ResolvedDefinition) {
				return true
			}
		}
	}
	return false
}

// isMethodImplParameter reports whether symbol is bound by the parameter
// vector of a method implemented in deftype, defrecord, reify, proxy, or an
// extend form. The implemented interface defines what the method may do with
// its arguments, e.g. Collection.toArray(T[]) must fill the array it receives.
func isMethodImplParameter(symbol *reader.RichNode, context map[string]interface{}) bool {
	if symbol == nil || symbol.ResolvedDefinition == nil {
		return false
	}
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for index := 1; index < len(ancestors); index++ {
		method, host := ancestors[index], ancestors[index-1]
		if method == nil || method.Type != reader.NodeList || len(method.Children) < 2 ||
			method.Children[0] == nil || method.Children[0].Type != reader.NodeSymbol ||
			host == nil || host.Type != reader.NodeList || len(host.Children) == 0 || host.Children[0] == nil {
			continue
		}
		switch strings.TrimPrefix(host.Children[0].Value, "clojure.core/") {
		case "deftype", "defrecord", "reify", "proxy", "extend-type", "extend-protocol", "specify", "specify!":
			if functionParamsContainNode(method, symbol.ResolvedDefinition) {
				return true
			}
		}
	}
	return false
}

func functionParamsContainNode(form, target *reader.RichNode) bool {
	for _, child := range form.Children[1:] {
		if child == nil {
			continue
		}
		if child.Type == reader.NodeVector {
			return nodeWithin(child, target)
		}
		if child.Type == reader.NodeList && len(child.Children) > 0 && child.Children[0] != nil &&
			child.Children[0].Type == reader.NodeVector && nodeWithin(child.Children[0], target) {
			return true
		}
	}
	return false
}

func nodeWithin(node, target *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node == target {
		return true
	}
	for _, child := range node.Children {
		if nodeWithin(child, target) {
			return true
		}
	}
	return false
}

func mutatedArrayOrigin(target *reader.RichNode, context map[string]interface{}) arrayOrigin {
	if target == nil || target.Type != reader.NodeSymbol {
		return arrayOriginUnknown
	}
	if target.Resolution != nil && !target.Resolution.Lexical {
		// A namespace Var: global state.
		return arrayOriginExternal
	}
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for index := len(ancestors) - 1; index >= 0; index-- {
		form := ancestors[index]
		if form == nil || form.Type != reader.NodeList || len(form.Children) < 2 || form.Children[0] == nil {
			continue
		}
		head := strings.TrimPrefix(form.Children[0].Value, "clojure.core/")
		switch head {
		case "let", "let*", "loop", "loop*", "when-let", "if-let", "when-some", "if-some", "binding":
			vector := form.Children[1]
			if vector == nil || vector.Type != reader.NodeVector {
				continue
			}
			for pair := len(vector.Children) - 2; pair >= 0; pair -= 2 {
				if name := vector.Children[pair]; name != nil && name.Type == reader.NodeSymbol && name.Value == target.Value {
					initializer := vector.Children[pair+1]
					if constructsLocalArray(initializer) {
						return arrayOriginLocalConstruction
					}
					if initializer != nil && initializer.Type == reader.NodeSymbol {
						if initializer.Resolution == nil {
							return arrayOriginUnknown
						}
						if initializer.Resolution.Kind == reader.ResolutionLocal {
							if isNamedFunctionParameter(initializer, context) {
								return arrayOriginExternal
							}
							return arrayOriginUnknown
						}
						return arrayOriginExternal
					}
					return arrayOriginUnknown
				}
			}
		case "defn", "defn-", "defmethod":
			// The caller's array crosses the public function boundary.
			if functionParamsBindSymbol(form, target.Value) {
				return arrayOriginExternal
			}
		case "fn", "fn*":
			// An anonymous fn parameter is often a reduce accumulator over an
			// array built nearby; its ownership is not proven.
			if functionParamsBindSymbol(form, target.Value) {
				return arrayOriginUnknown
			}
		}
	}
	return arrayOriginUnknown
}

func functionParamsBindSymbol(form *reader.RichNode, name string) bool {
	for _, child := range form.Children[1:] {
		if child == nil {
			continue
		}
		if child.Type == reader.NodeVector {
			return bindingPatternHas(child, name)
		}
		if child.Type == reader.NodeList && len(child.Children) > 0 && child.Children[0] != nil && child.Children[0].Type == reader.NodeVector {
			if bindingPatternHas(child.Children[0], name) {
				return true
			}
		}
	}
	return false
}

func bindingPatternHas(node *reader.RichNode, name string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol {
		return node.Value == name
	}
	for _, child := range node.Children {
		if bindingPatternHas(child, name) {
			return true
		}
	}
	return false
}

func isProvenArrayMutation(node *reader.RichNode, context map[string]interface{}) bool {
	if node == nil || len(node.Children) < 2 || !isLocalRuntimeScope(context) ||
		!isKnownCoreMutation(node,
			"clojure.core/aset", "clojure.core/aset-boolean", "clojure.core/aset-byte",
			"clojure.core/aset-char", "clojure.core/aset-double", "clojure.core/aset-float",
			"clojure.core/aset-int", "clojure.core/aset-long", "clojure.core/aset-short") {
		return false
	}
	target := node.Children[1]
	return target != nil && target.Type == reader.NodeSymbol && isJavaArrayTypeHint(target.TypeHint)
}

func isGeneratedOrMacroCode(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil {
			continue
		}
		switch ancestor.Type {
		case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
			return true
		}
		if ancestor.Type == reader.NodeList && len(ancestor.Children) > 0 &&
			ancestor.Children[0].Type == reader.NodeSymbol && ancestor.Children[0].Value == "defmacro" {
			return true
		}
	}
	return false
}

// case constants are data, even when a grouped constant is represented by a
// list whose first symbol happens to be `def` or `defonce`.
func isCaseConstantPosition(context map[string]interface{}) bool {
	value, _ := context["isInCaseConstantPosition"].(bool)
	return value
}

func isValidAlterVarRootCall(node *reader.RichNode) bool {
	return node != nil && len(node.Children) >= 3 &&
		rules.CallResolvesTo(node, "clojure.core/alter-var-root")
}

func isNonExecutableContext(context map[string]interface{}) bool {
	execution := rules.CurrentExecutionContext(context)
	return execution == rules.ExecutionNonEvaluated || execution == rules.ExecutionUnknown
}

func (r *ImmutabilityViolationRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 ||
		strings.HasSuffix(filepath, "user.clj") || strings.Contains(filepath, "/dev/") {
		return nil
	}

	if isGeneratedOrMacroCode(context) {
		return nil
	}
	if isNonExecutableContext(context) || r.IsInside(context, "__non-evaluated__", "comment") {
		return nil
	}
	if isCaseConstantPosition(context) {
		return nil
	}

	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return nil
	}

	if head.Resolution != nil && head.Resolution.Kind == reader.ResolutionLocal {
		return nil
	}

	symValue := head.Value
	unqualifiedSym := symValue
	if slash := strings.Index(symValue, "/"); slash >= 0 {
		unqualifiedSym = symValue[slash+1:]
	}

	if isKnownCoreMutation(node, "clojure.core/def", "clojure.core/defonce") &&
		isLocalRuntimeScope(context) && !argumentOfUnknownForm(context) {
		return &rules.Finding{
			RuleID:     r.ID,
			Message:    fmt.Sprintf("Found `%s` inside a local scope. This mutates global state and should be avoided.", unqualifiedSym),
			Filepath:   filepath,
			Location:   node.Location,
			Severity:   r.Severity,
			Confidence: rules.ConfidenceProven,
		}
	}

	if isValidAlterVarRootCall(node) {
		finding := &rules.Finding{
			RuleID:   r.ID,
			Message:  "Found alter-var-root mutating a Var during runtime. Review whether this global state transition is required and whether its ownership and concurrency contract are explicit.",
			Filepath: filepath,
			Location: head.Location,
			Severity: r.Severity,
			Tags:     []string{"state-mutation", "contract-dependent"},
		}
		return rules.SetContextualFindingWithEvidence(
			finding,
			"The mutation is structurally proven, but intent, ownership, lifecycle, and concurrency requirements are not visible at the call site.",
			"ownership", "lifecycle", "concurrency-contract",
		)
	}

	if isKnownCoreMutation(node, "clojure.core/ref-set") {
		insideDosync, _ := context["isInsideDosync"].(bool)
		if !insideDosync {
			insideFunction, _ := context["isInsideFunction"].(bool)
			severity := r.Severity
			tags := []string(nil)
			if insideFunction {
				severity = rules.SeverityHint
				tags = []string{"low-confidence", "external-transaction-unknown"}
			}
			finding := &rules.Finding{
				RuleID:     r.ID,
				Message:    "Found `ref-set` outside of `dosync`. Use `dosync` to ensure transactional safety with refs.",
				Filepath:   filepath,
				Location:   head.Location,
				Severity:   severity,
				Confidence: rules.ConfidenceProven,
				Tags:       tags,
			}
			if insideFunction {
				return rules.SetContextualFindingWithEvidence(finding, "The transactional scope may be established by an external contract that is not visible in this file.", "transactional-scope", "external-transaction-boundary")
			}
			return finding
		}
	}

	if isProvenArrayMutation(node, context) {
		finding := &rules.Finding{
			RuleID:     r.ID,
			Message:    "Found direct mutation of a type-hinted Java array inside a local scope. Prefer returning a new value or document the required mutable boundary.",
			Filepath:   filepath,
			Location:   head.Location,
			Severity:   r.Severity,
			Confidence: rules.ConfidenceProven,
		}
		switch mutatedArrayOrigin(node.Children[1], context) {
		case arrayOriginLocalConstruction:
			// Filling an array built in the same scope mutates no global or
			// externally visible state.
			return nil
		case arrayOriginUnknown:
			if isMethodImplParameter(node.Children[1], context) {
				return nil
			}
			return rules.SetContextualFindingWithEvidence(finding, "The array may be local construction or shared state; its ownership is not proven.", "array-ownership")
		}
		return finding
	}

	return nil
}

func init() {
	rules.RegisterRule(&ImmutabilityViolationRule{Rule: rules.Rule{
		ID:                    "immutability-violation",
		Name:                  "Immutability Violation",
		Description:           "Detects direct state mutation and violations of functional purity when the mutation is structurally resolved. Contextual findings identify real mutations whose ownership, lifecycle, or concurrency contract is not locally provable.",
		ContextualDescription: "It may be contextual when alter-var-root is used deliberately for configuration, tooling, REPL lifecycle, or another explicit global-state boundary.",
		Severity:              rules.SeverityWarning,
	}})
}
