package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type DirectClojureLangRTRule struct {
	Rule
	AllowedFunctions []string `json:"allowed_functions" yaml:"allowed_functions"`
}

func (r *DirectClojureLangRTRule) Meta() Rule {
	return r.Rule
}

func (r *DirectClojureLangRTRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 1 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return nil
	}

	if !r.isClojureLangRTCall(firstChild.Value) {
		return nil
	}

	rtFunction := r.extractRTFunction(firstChild.Value)

	if r.isAllowedFunction(rtFunction) {
		return nil
	}

	suggestion := r.getSuggestionForFunction(rtFunction)

	message := fmt.Sprintf(
		"Direct usage of clojure.lang.RT detected: '%s'. "+
			"clojure.lang.RT is an internal API and its usage should be avoided. %s",
		firstChild.Value,
		suggestion,
	)

	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func (r *DirectClojureLangRTRule) isClojureLangRTCall(symbol string) bool {

	if strings.HasPrefix(symbol, "clojure.lang.RT/") {
		return true
	}

	if strings.HasPrefix(symbol, "RT/") {

		return true
	}

	return false
}

func (r *DirectClojureLangRTRule) extractRTFunction(symbol string) string {
	if strings.Contains(symbol, "/") {
		parts := strings.Split(symbol, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}
	return symbol
}

func (r *DirectClojureLangRTRule) isAllowedFunction(function string) bool {
	for _, allowed := range r.AllowedFunctions {
		if function == allowed {
			return true
		}
	}
	return false
}

func (r *DirectClojureLangRTRule) getSuggestionForFunction(function string) string {
	suggestions := map[string]string{
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

	if suggestion, found := suggestions[function]; found {
		return suggestion
	}

	return "Prefer using Clojure's standard library functions instead of accessing RT directly."
}

func init() {
	defaultRule := &DirectClojureLangRTRule{
		Rule: Rule{
			ID:          "direct-use-of-clojure-lang-rt",
			Name:        "Direct Use of clojure.lang.RT",
			Description: "Detects direct usage of clojure.lang.RT internal API. Direct usage of clojure.lang.RT should be avoided as it's an internal implementation detail that may change between Clojure versions. Use the standard library functions instead.",
			Severity:    SeverityWarning,
		},
		AllowedFunctions: []string{},
	}

	RegisterRule(defaultRule)
}
