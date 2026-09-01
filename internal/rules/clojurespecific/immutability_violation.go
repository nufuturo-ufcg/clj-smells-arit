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
	switch hint {
	case "boolean", "booleans", "byte", "bytes", "char", "chars",
		"double", "doubles", "float", "floats", "int", "ints",
		"long", "longs", "object", "objects", "short", "shorts":
		return true
	default:
		return false
	}
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
		isLocalRuntimeScope(context) {
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Found `%s` inside a local scope. This mutates global state and should be avoided.", unqualifiedSym),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
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
				RuleID:   r.ID,
				Message:  "Found `ref-set` outside of `dosync`. Use `dosync` to ensure transactional safety with refs.",
				Filepath: filepath,
				Location: head.Location,
				Severity: severity,
				Tags:     tags,
			}
			if insideFunction {
				return rules.SetContextualFindingWithEvidence(finding, "The transactional scope may be established by an external contract that is not visible in this file.", "transactional-scope", "external-transaction-boundary")
			}
			return finding
		}
	}

	if isProvenArrayMutation(node, context) {
		return &rules.Finding{
			RuleID:   r.ID,
			Message:  "Found direct mutation of a type-hinted Java array inside a local scope. Prefer returning a new value or document the required mutable boundary.",
			Filepath: filepath,
			Location: head.Location,
			Severity: r.Severity,
		}
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
