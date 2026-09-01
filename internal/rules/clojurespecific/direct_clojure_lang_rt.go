package clojurespecific

import (
	"fmt"
	"github.com/thlaurentino/arit/internal/rules"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

var rtSuggestions = map[string]string{
	"iter":      "Consider using (seq coll) or direct iteration with doseq/for instead of RT/iter.",
	"get":       "Use (get map key) or (map key) instead of RT/get.",
	"assoc":     "Use (assoc map key val) instead of RT/assoc.",
	"conj":      "Use (conj coll item) instead of RT/conj.",
	"count":     "Use (count coll) instead of RT/count.",
	"nth":       "Use (nth coll index) instead of RT/nth.",
	"first":     "Use (first coll) instead of RT/first.",
	"rest":      "Use (rest coll) instead of RT/rest.",
	"seq":       "Use (seq coll) instead of RT/seq.",
	"cons":      "Use (cons item coll) instead of RT/cons.",
	"empty":     "Use (empty coll) instead of RT/empty.",
	"meta":      "Use (meta obj) instead of RT/meta.",
	"with-meta": "Use (with-meta obj meta) instead of RT/withMeta.",
	"print":     "Use (print obj) or (println obj) instead of RT print functions.",
	"load":      "Use (load filename) or (require) instead of RT/load.",
	"var":       "Use (var symbol) or #'symbol instead of RT/var.",
	"deref":     "Use (deref ref) or @ref instead of RT/deref.",
}

func init() {
	rules.NewRule("direct-use-of-clojure-lang-rt").
		Name("Direct Use of clojure.lang.RT").
		Description("Detects direct usage of clojure.lang.RT internal API. Direct usage of clojure.lang.RT should be avoided as it's an internal implementation detail that may change between Clojure versions. Use the standard library functions instead.").Severity(rules.SeverityWarning).
		When(rules.IsList()).
		When(rules.HasMinChildren(1)).
		When(rules.ChildIsSymbol(0)).
		When(func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
			if rules.IsPathAllowed(context, "direct-use-of-clojure-lang-rt", filepath) {
				return false
			}
			if rules.CurrentExecutionContext(context) == rules.ExecutionNonEvaluated {
				return false
			}
			if directRTIsNonEvaluated(context) {
				return false
			}

			head := node.Children[0]
			resolution := head.Resolution
			if resolution == nil || resolution.Kind != reader.ResolutionJavaStatic ||
				!strings.HasPrefix(resolution.CanonicalName, "clojure.lang.RT/") {
				return false
			}

			sym := head.Value

			parts := strings.Split(sym, "/")
			rtFunc := parts[len(parts)-1]
			if len(parts) > 1 && rules.MatchesConfiguredName(parts[0], rules.RuleSettingStringSlice(context, "direct-use-of-clojure-lang-rt", "allowed_namespaces")) {
				return false
			}

			allowed := rules.GetConfigStringSlice(context, "direct-use-of-clojure-lang-rt", "allowed_functions")
			for _, fn := range allowed {
				if fn == rtFunc {
					return false
				}
			}
			return true
		}).
		ContextualWithEvidence("Direct RT usage is structurally detectable, but it may be required for classloaders, runtime behavior, compatibility, or performance.", "runtime-compatibility", "classloader-contract", "performance-contract").
		SeverityFunc(func(_ *reader.RichNode, context map[string]interface{}, defaultSeverity rules.Severity) rules.Severity {
			return rules.ContextualSeverity(context, defaultSeverity)
		}).
		MessageFunc(func(node *reader.RichNode, _ map[string]interface{}) string {
			sym := node.Children[0].Value
			parts := strings.Split(sym, "/")
			rtFunc := parts[len(parts)-1]

			suggestion := "Prefer using Clojure's standard library functions instead of accessing RT directly."
			if sugg, found := rtSuggestions[rtFunc]; found {
				suggestion = sugg
			}

			return fmt.Sprintf(
				"Direct usage of clojure.lang.RT detected: '%s'. "+
					"clojure.lang.RT is an internal API and its usage should be avoided. %s",
				sym,
				suggestion,
			)
		}).
		Register()
}

func directRTIsNonEvaluated(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	syntaxQuoteDepth := 0
	insideMacroDefinition := false
	for _, ancestor := range ancestors {
		if ancestor == nil {
			continue
		}
		if ancestor.Type == reader.NodeList && len(ancestor.Children) > 0 && ancestor.Children[0] != nil &&
			ancestor.Children[0].Type == reader.NodeSymbol && ancestor.Children[0].Value == "defmacro" {
			insideMacroDefinition = true
		}
		switch ancestor.Type {
		case reader.NodeQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
			return true
		case reader.NodeSyntaxQuote:
			syntaxQuoteDepth++
		case reader.NodeUnquote, reader.NodeUnquoteSplice:
			if syntaxQuoteDepth > 0 {
				syntaxQuoteDepth--
			}
		}
	}
	return syntaxQuoteDepth > 0 && !insideMacroDefinition
}
