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

func (r *RelyingOnLoadTimeSideEffectsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if !rules.ExecutesAtLoad(context) || node == nil || node.Type != reader.NodeList ||
		len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol ||
		!loadTimeEffectOperation(node) {
		return nil
	}
	if !r.IsInside(context, "def", "defonce") || r.IsInside(context, "ns") {
		return nil
	}
	return &rules.Finding{
		RuleID: r.ID, Filepath: filepath, Location: node.Location, Severity: r.Severity,
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
