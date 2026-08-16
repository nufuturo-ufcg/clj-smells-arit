package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type UnnecessaryMacrosRule struct{ rules.Rule }

func (r *UnnecessaryMacrosRule) Meta() rules.Rule { return r.Rule }

func macroContainsUnsafeFeature(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeUnquoteSplice {
		return true
	}
	if node.Type == reader.NodeNil {
		return true
	}
	if node.Type == reader.NodeSymbol {
		if strings.Contains(node.Value, "clojure.lang.RT") || strings.Contains(node.Value, "core.async") {
			return true
		}
		switch node.Value {
		case "&form", "&env", "gensym", "macroexpand", "macroexpand-1", "eval", "let", "let*", "loop", "recur", "fn", "fn*", "if", "if-not", "if-let", "if-some", "when", "when-not", "when-let", "when-some", "cond", "condp", "try", "catch", "finally", "binding", "set!", "case", "go", "go-loop", "thread", "slurp", "spit":
			return true
		}
	}
	for _, child := range node.Children {
		if macroContainsUnsafeFeature(child) {
			return true
		}
	}
	return false
}

func macroCountUnquotedParameters(node *reader.RichNode, parameters map[string]int, insideUnquote bool) {
	if node == nil {
		return
	}
	if node.Type == reader.NodeUnquote {
		insideUnquote = true
	}
	if insideUnquote && node.Type == reader.NodeSymbol {
		if _, exists := parameters[node.Value]; exists {
			parameters[node.Value]++
		}
	}
	for _, child := range node.Children {
		macroCountUnquotedParameters(child, parameters, insideUnquote)
	}
}

func macroHasSyntaxQuote(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSyntaxQuote {
		return true
	}
	for _, child := range node.Children {
		if macroHasSyntaxQuote(child) {
			return true
		}
	}
	return false
}

func (r *UnnecessaryMacrosRule) Check(node *reader.RichNode, _ map[string]interface{}, filepath string) *rules.Finding {
	if coreAsyncOperation(node) != "defmacro" || len(node.Children) < 4 || node.Children[1].Type != reader.NodeSymbol {
		return nil
	}
	body := node.Children[len(node.Children)-1]
	if !macroHasSyntaxQuote(body) || macroContainsUnsafeFeature(body) {
		return nil
	}
	params := node.Children[2]
	if params.Type != reader.NodeVector {
		return nil
	}
	parameterUses := make(map[string]int)
	for _, param := range params.Children {
		if param.Type == reader.NodeSymbol && param.Value != "&" {
			parameterUses[param.Value] = 0
		}
	}
	if len(parameterUses) == 0 {
		return nil
	}
	macroCountUnquotedParameters(body, parameterUses, false)
	for _, uses := range parameterUses {
		if uses != 1 {
			return nil
		}
	}
	name := node.Children[1].Value
	return &rules.Finding{
		RuleID: r.ID, Filepath: filepath, Location: node.Location, Severity: r.Severity,
		Message: fmt.Sprintf("Macro %q only wraps function-like evaluation; prefer a normal function for composability.", name),
	}
}

func init() {
	rules.RegisterRule(&UnnecessaryMacrosRule{Rule: rules.Rule{
		ID: "unnecessary-macros", Name: "Unnecessary Macros",
		Description: "Detects simple syntax-quoted macros with no compile-time or evaluation-control feature.",
		Severity:    rules.SeverityHint,
	}})
}
