package rules

import (
	"fmt"
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
	NewRule("direct-use-of-clojure-lang-rt").
		Name("Direct Use of clojure.lang.RT").
		Description("Detects direct usage of clojure.lang.RT internal API. Direct usage of clojure.lang.RT should be avoided as it's an internal implementation detail that may change between Clojure versions. Use the standard library functions instead.").
		Severity(SeverityWarning).
		When(IsList()).
		When(HasMinChildren(1)).
		When(ChildIsSymbol(0)).
		When(func(node *reader.RichNode, context map[string]interface{}, _ string) bool {
			sym := node.Children[0].Value
			if !strings.HasPrefix(sym, "clojure.lang.RT/") && !strings.HasPrefix(sym, "RT/") {
				return false
			}

			parts := strings.Split(sym, "/")
			rtFunc := parts[len(parts)-1]

			allowed := GetConfigStringSlice(context, "direct-use-of-clojure-lang-rt", "allowed_functions")
			for _, fn := range allowed {
				if fn == rtFunc {
					return false
				}
			}
			return true
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
