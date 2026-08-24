package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type MisuseOfDynamicScopeRule struct {
	rules.Rule
}

func (r *MisuseOfDynamicScopeRule) Meta() rules.Rule {
	return r.Rule
}

func isAllowedDynamicVar(name string) bool {
	switch name {
	case "*out*", "*err*", "*in*", "*ns*", "*warn-on-reflection*", "*file*", "*compile-path*", "*command-line-args*", "*agent*", "*math-context*", "*print-length*", "*print-level*", "*data-readers*", "*default-data-reader-fn*", "*read-eval*":
		return true
	}
	return false
}

func isExplicitResourceVar(name string) bool {
	lower := strings.ToLower(name)
	lower = strings.TrimPrefix(lower, "*")
	lower = strings.TrimSuffix(lower, "*")

	parts := strings.Split(lower, "-")
	keywords := map[string]bool{
		"db": true, "conn": true, "pool": true, "tx": true, "transaction": true,
		"client": true, "producer": true, "socket": true, "ds": true,
		"redis": true, "kafka": true, "s3": true, "http": true,
		"writer": true, "session": true,
	}

	for _, part := range parts {
		if keywords[part] {
			return true
		}
	}
	return false
}

func (r *MisuseOfDynamicScopeRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol {
		return nil
	}
	if !hasAsyncBoundary(context) {
		return nil
	}
	allowedVars := rules.RuleSettingStringSlice(context, r.Meta().ID, "allowed_vars")

	if firstElement.Value == "def" && len(node.Children) > 1 {
		var name string
		var loc *reader.Location

		for i := 1; i < len(node.Children); i++ {
			if node.Children[i].Type == reader.NodeSymbol {
				name = node.Children[i].Value
				loc = node.Children[i].Location
				break
			}
		}

		if name != "" && strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") && len(name) > 2 {
			if !isAllowedDynamicVar(name) && !isExplicitResourceVar(name) && !rules.MatchesConfiguredName(name, allowedVars) {
				return &rules.Finding{
					RuleID:   r.Meta().ID,
					Message:  fmt.Sprintf("Defining custom dynamic variable `%s`. Dynamic scope is often misused for passing business context, which obfuscates data flow and causes bugs in async boundaries.", name),
					Filepath: filepath,
					Location: loc,
					Severity: rules.ContextualSeverity(context, r.Meta().Severity),
					Tags:     rules.ContextualTags(context),
				}
			}
		}
	}

	// A binding created inside a worker is local configuration for that worker;
	// it does not by itself prove that a parent binding was lost. The previous
	// implementation reported every binding below future/send, which produced
	// false positives for explicit worker-local configuration. Keep the check
	// only for core.async state-machine boundaries, where parking can resume
	// execution outside the lexical binding scope.
	if firstElement.Value == "binding" && len(node.Children) > 1 {
		if !hasProblematicAsyncBoundary(context) {
			return nil
		}
		bindings := node.Children[1]
		if bindings.Type == reader.NodeVector {
			for i := 0; i < len(bindings.Children); i += 2 {
				varNode := bindings.Children[i]
				if varNode.Type == reader.NodeSymbol {
					name := varNode.Value
					if !isAllowedDynamicVar(name) && !isExplicitResourceVar(name) && !rules.MatchesConfiguredName(name, allowedVars) {
						severity := rules.ContextualSeverity(context, r.Meta().Severity)
						tags := rules.ContextualTags(context)
						if i+1 < len(bindings.Children) && bindings.Children[i+1].Type == reader.NodeNil {
							severity = rules.SeverityHint
							tags = append(tags, "explicit-reset")
						}
						return &rules.Finding{
							RuleID:   r.Meta().ID,
							Message:  fmt.Sprintf("Binding custom dynamic variable `%s` across a core.async boundary. Prefer explicit data flow or preserve the binding around the complete operation.", name),
							Filepath: filepath,
							Location: varNode.Location,
							Severity: severity,
							Tags:     tags,
						}
					}
				}
			}
		}
	}

	return nil
}

func hasAsyncBoundary(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	for _, form := range enclosing {
		switch strings.TrimPrefix(form, "clojure.core/") {
		case "go", "go-loop", "thread", "future", "future-call", "agent", "send", "send-off", "pmap":
			return true
		}
	}
	return false
}

func hasProblematicAsyncBoundary(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	for _, form := range enclosing {
		form = strings.TrimPrefix(form, "clojure.core/")
		switch {
		case form == "go" || form == "go-loop" || strings.HasSuffix(form, "/go") || strings.HasSuffix(form, "/go-loop"):
			return true
		}
	}
	return false
}

func init() {
	defaultRule := &MisuseOfDynamicScopeRule{
		Rule: rules.Rule{
			ID:          "misuse-of-dynamic-scope",
			Name:        "Misuse of Dynamic Scope",
			Description: "Flags the definition and usage of custom dynamic variables (*var*), which are often misused to pass business parameters implicitly, hiding dependencies and breaking thread boundaries.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
