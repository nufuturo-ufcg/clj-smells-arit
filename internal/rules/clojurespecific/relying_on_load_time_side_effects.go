package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type RelyingOnLoadTimeSideEffectsRule struct{ rules.Rule }

func (r *RelyingOnLoadTimeSideEffectsRule) Meta() rules.Rule { return r.Rule }

func loadTimeEffectOperation(node *reader.RichNode) bool {
	exact := map[string]struct{}{
		"clojure.core/slurp":    {},
		"clojure.core/spit":     {},
		"clojure.java.shell/sh": {}, "shell/sh": {},
		".mkdirs":          {},
		"java.net.Socket.": {}, "Socket.": {},
		"com.zaxxer.hikari.HikariDataSource.": {}, "HikariDataSource.": {},
	}
	resolved := rules.ResolvedCall(node)
	if resolved == nil {
		return false
	}
	_, ok := exact[resolved.CanonicalName]
	return ok
}

func isStaticClasspathResourceRead(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if len(node.Children) == 2 && rules.CallResolvesTo(node, "clojure.core/slurp") {
		resource := node.Children[1]
		return resource != nil && resource.Type == reader.NodeList && rules.CallResolvesTo(resource, "clojure.java.io/resource")
	}

	// Threading macros preserve the same static-resource proof as the direct
	// form `(slurp (io/resource ...))`. Without this normalization, the rule
	// reported classpath reads written idiomatically as `(-> (resource ...) slurp)`.
	if node.Type != reader.NodeList || len(node.Children) < 3 || node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	thread := node.Children[0].Value
	if thread != "->" && thread != "->>" {
		return false
	}
	if node.Children[1].Type != reader.NodeList || !rules.CallResolvesTo(node.Children[1], "clojure.java.io/resource") {
		return false
	}
	for _, step := range node.Children[2:] {
		if step.Type == reader.NodeSymbol {
			if step.Resolution != nil && step.Resolution.CanonicalName == "clojure.core/slurp" {
				return true
			}
			if step.Value == "slurp" && (step.Resolution == nil || step.Resolution.Kind == reader.ResolutionUnresolved) {
				return true
			}
		}
		if step.Type == reader.NodeList && rules.CallResolvesTo(step, "clojure.core/slurp") {
			return true
		}
	}
	return false
}

func (r *RelyingOnLoadTimeSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if !rules.ExecutesAtLoad(context) || node == nil || node.Type != reader.NodeList ||
		len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol ||
		!loadTimeEffectOperation(node) {
		return nil
	}
	if !r.IsInside(context, "def", "defonce") || r.IsInside(context, "ns") {
		return nil
	}
	if isStaticClasspathResourceRead(node) {
		return nil
	}
	return &rules.Finding{
		RuleID: r.ID, Filepath: filepath, Location: node.Location,
		Severity: rules.ContextualSeverity(context, r.Severity), Tags: rules.ContextualTags(context),
		Message: fmt.Sprintf("Side-effecting operation %q runs while the namespace is loaded; defer it to application startup.", node.Children[0].Value),
	}
}

func init() {
	rules.RegisterRule(&RelyingOnLoadTimeSideEffectsRule{Rule: rules.Rule{
		ID: "relying-on-load-time-side-effects", Name: "Relying on Load-Time Side Effects",
		Description: "Detects known I/O, network, process, and resource initialization inside top-level vars.",
		Severity:    rules.SeverityWarning,
	}})
}
