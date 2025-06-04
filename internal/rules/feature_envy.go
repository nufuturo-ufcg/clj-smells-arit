package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type FeatureEnvyRule struct {
	Rule
	EnvyThreshold    float64  `json:"envy_threshold" yaml:"envy_threshold"`
	MinExternalCalls int      `json:"min_external_calls" yaml:"min_external_calls"`
	MinTotalCalls    int      `json:"min_total_calls" yaml:"min_total_calls"`
	IgnoreNamespaces []string `json:"ignore_namespaces" yaml:"ignore_namespaces"`
}

type CallAnalysis struct {
	FunctionName    string
	CurrentNS       string
	InternalCalls   int
	ExternalCalls   int
	ExternalTargets map[string]int
	TotalCalls      int
	EnvyRatio       float64
}

func (r *FeatureEnvyRule) Meta() Rule {
	return r.Rule
}

func (r *FeatureEnvyRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if !r.isFunctionDefinition(node) {
		return nil
	}

	funcName := r.extractFunctionName(node)
	if funcName == "" {
		return nil
	}

	currentNS := r.extractCurrentNamespace(context, filepath)

	analysis := r.analyzeFunctionCalls(node, currentNS)
	analysis.FunctionName = funcName

	if !r.meetsMinimumCriteria(analysis) {
		return nil
	}

	analysis.EnvyRatio = float64(analysis.ExternalCalls) / float64(analysis.TotalCalls)

	if analysis.EnvyRatio >= r.EnvyThreshold {
		return r.createFinding(analysis, filepath, node)
	}

	return nil
}

func (r *FeatureEnvyRule) isFunctionDefinition(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	return firstChild.Value == "defn" || firstChild.Value == "defn-"
}

func (r *FeatureEnvyRule) extractFunctionName(node *reader.RichNode) string {
	if len(node.Children) >= 2 && node.Children[1].Type == reader.NodeSymbol {
		return node.Children[1].Value
	}
	return ""
}

func (r *FeatureEnvyRule) extractCurrentNamespace(context map[string]interface{}, filepath string) string {

	parts := strings.Split(filepath, "/")
	if len(parts) > 0 {
		filename := parts[len(parts)-1]
		if strings.HasSuffix(filename, ".clj") || strings.HasSuffix(filename, ".cljs") || strings.HasSuffix(filename, ".cljc") {

			nsName := strings.TrimSuffix(filename, ".clj")
			nsName = strings.TrimSuffix(nsName, ".cljs")
			nsName = strings.TrimSuffix(nsName, ".cljc")
			nsName = strings.ReplaceAll(nsName, "_", "-")
			return nsName
		}
	}

	return "unknown"
}

func (r *FeatureEnvyRule) analyzeFunctionCalls(node *reader.RichNode, currentNS string) *CallAnalysis {
	analysis := &CallAnalysis{
		CurrentNS:       currentNS,
		InternalCalls:   0,
		ExternalCalls:   0,
		ExternalTargets: make(map[string]int),
		TotalCalls:      0,
	}

	r.walkFunctionCalls(node, analysis)

	return analysis
}

func (r *FeatureEnvyRule) walkFunctionCalls(node *reader.RichNode, analysis *CallAnalysis) {
	if node == nil {
		return
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 {
		firstChild := node.Children[0]
		if firstChild.Type == reader.NodeSymbol {
			r.analyzeCall(firstChild.Value, analysis)
		}
	}

	for _, child := range node.Children {
		r.walkFunctionCalls(child, analysis)
	}
}

func (r *FeatureEnvyRule) analyzeCall(call string, analysis *CallAnalysis) {

	if r.isSpecialForm(call) {
		return
	}

	analysis.TotalCalls++

	if strings.Contains(call, "/") {

		parts := strings.SplitN(call, "/", 2)
		namespace := parts[0]

		if r.shouldIgnoreNamespace(namespace) {
			return
		}

		if namespace != analysis.CurrentNS {
			analysis.ExternalCalls++
			analysis.ExternalTargets[namespace]++
		} else {
			analysis.InternalCalls++
		}
	} else {

		analysis.InternalCalls++
	}
}

func (r *FeatureEnvyRule) isSpecialForm(call string) bool {
	specialForms := map[string]bool{
		"let": true, "if": true, "when": true, "cond": true, "case": true,
		"do": true, "loop": true, "recur": true, "fn": true, "quote": true,
		"var": true, "def": true, "set!": true, "monitor-enter": true,
		"monitor-exit": true, "throw": true, "try": true, "catch": true,
		"finally": true, "new": true, ".": true,
	}

	return specialForms[call]
}

func (r *FeatureEnvyRule) shouldIgnoreNamespace(namespace string) bool {
	for _, ignored := range r.IgnoreNamespaces {
		if namespace == ignored {
			return true
		}
	}
	return false
}

func (r *FeatureEnvyRule) meetsMinimumCriteria(analysis *CallAnalysis) bool {
	if analysis.TotalCalls < r.MinTotalCalls {
		return false
	}

	if analysis.ExternalCalls < r.MinExternalCalls {
		return false
	}

	return true
}

func (r *FeatureEnvyRule) createFinding(analysis *CallAnalysis, filepath string, node *reader.RichNode) *Finding {

	maxNamespace := ""
	maxCount := 0
	for ns, count := range analysis.ExternalTargets {
		if count > maxCount {
			maxCount = count
			maxNamespace = ns
		}
	}

	suggestion := r.generateSuggestion(analysis, maxNamespace)

	message := fmt.Sprintf(
		"Function '%s' shows feature envy (%.1f%% external calls: %d external vs %d internal). %s",
		analysis.FunctionName,
		analysis.EnvyRatio*100,
		analysis.ExternalCalls,
		analysis.InternalCalls,
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

func (r *FeatureEnvyRule) generateSuggestion(analysis *CallAnalysis, enviedNamespace string) string {
	if enviedNamespace != "" && analysis.ExternalTargets[enviedNamespace] > analysis.InternalCalls {
		return fmt.Sprintf(
			"Consider moving this function to namespace '%s' where it primarily operates, or refactor to reduce external dependencies.",
			enviedNamespace,
		)
	}

	return "Consider refactoring to reduce dependencies on external namespaces, or move this function to where its data/operations primarily reside."
}

func init() {
	defaultRule := &FeatureEnvyRule{
		Rule: Rule{
			ID:          "feature-envy",
			Name:        "Feature Envy",
			Description: "Detects functions that make more calls to other namespaces than to their own, indicating they might be in the wrong place. Based on the classic 'Feature Envy' code smell adapted for functional programming.",
			Severity:    SeverityWarning,
		},
		EnvyThreshold:    0.7,
		MinExternalCalls: 3,
		MinTotalCalls:    5,
		IgnoreNamespaces: []string{
			"clojure.core", "cljs.core", "clojure.string", "clojure.set",
			"clojure.walk", "clojure.zip", "clojure.data", "clojure.edn",
			"clojure.java.io", "clojure.pprint", "clojure.repl",
		},
	}

	RegisterRule(defaultRule)
}
