package rules

import (
	"path/filepath"
	"strings"

	"github.com/thlaurentino/arit/internal/config"
)

// MarkContextualFinding normalizes contextuality for findings created by
// hand-written rules. Tags are kept for compatibility with existing reports,
// while the explicit field lets reporters filter without guessing from the
// severity (HINT is not synonymous with contextual).
func MarkContextualFinding(finding *Finding) {
	if finding == nil {
		return
	}
	if finding.Contextual {
		if finding.ContextualReason == "" {
			finding.ContextualReason = "A classificação depende do contexto do arquivo ou do contrato do código."
		}
		appendContextualTag(finding)
		return
	}
	for _, tag := range finding.Tags {
		switch tag {
		case "contextual", "contract-dependent", "producer-only", "external-transaction-unknown", "conditional-load":
			finding.Contextual = true
			if finding.ContextualReason == "" {
				finding.ContextualReason = "A classificação depende do contexto do arquivo ou do contrato do código."
			}
			appendContextualTag(finding)
			return
		}
	}
}

// SetContextualFinding marks one occurrence as a review candidate without
// changing the rule's ability to detect the underlying structural pattern.
// Rules should use this only when the evidence is insufficient to call the
// occurrence a real issue independently of its contract or lifecycle.
func SetContextualFinding(finding *Finding, reason string) *Finding {
	if finding == nil {
		return nil
	}
	finding.Contextual = true
	if reason != "" {
		finding.ContextualReason = reason
	}
	appendContextualTag(finding)
	return finding
}

func appendContextualTag(finding *Finding) {
	for _, tag := range finding.Tags {
		if tag == "contextual" {
			return
		}
	}
	finding.Tags = append(finding.Tags, "contextual")
}

func IsContextualFinding(finding *Finding) bool {
	if finding == nil {
		return false
	}
	if finding.Contextual {
		return true
	}
	for _, tag := range finding.Tags {
		switch tag {
		case "contextual", "contract-dependent", "producer-only", "external-transaction-unknown", "conditional-load":
			return true
		}
	}
	return false
}

// FileRole is supplied by the analyzer for rules that need to distinguish
// production code from examples, fixtures, generated output, and tooling.
func FileRole(context map[string]interface{}) string {
	if role, ok := context["file-role"].(string); ok {
		return role
	}
	return "production"
}

// ContextualSeverity keeps structurally relevant findings visible while
// lowering confidence for objectively non-production source roles.
func ContextualSeverity(context map[string]interface{}, defaultSeverity Severity) Severity {
	if IsContextualSource(context) {
		return SeverityHint
	}
	return defaultSeverity
}

func ContextualTags(context map[string]interface{}) []string {
	if !IsContextualSource(context) {
		return nil
	}
	return []string{"contextual", "low-confidence"}
}

func IsContextualSource(context map[string]interface{}) bool {
	if FileRole(context) != "production" {
		return true
	}
	namespace := strings.ToLower(CurrentNamespace(context))
	for _, marker := range []string{"clojure.", "cljs.", ".compiler", ".runtime", ".serialization", ".classloader", ".bootstrap"} {
		if strings.Contains(namespace, marker) || strings.HasPrefix(namespace, strings.TrimSuffix(marker, ".")) {
			return true
		}
	}
	return false
}

func CurrentNamespace(context map[string]interface{}) string {
	if namespace, ok := context["current-namespace"].(string); ok {
		return namespace
	}
	return ""
}

func IsNonRuntime(context map[string]interface{}) bool {
	switch CurrentExecutionContext(context) {
	case ExecutionNonEvaluated, ExecutionDeferred:
		return true
	default:
		return false
	}
}

func RuleSettingStringSlice(context map[string]interface{}, ruleID, key string) []string {
	cfg, ok := context["config"].(*config.Config)
	if !ok || cfg == nil || cfg.RuleConfig == nil {
		return nil
	}
	settings, ok := cfg.RuleConfig[ruleID]
	if !ok {
		return nil
	}
	value, ok := settings[key]
	if !ok {
		return nil
	}
	var values []string
	switch typed := value.(type) {
	case []string:
		values = append(values, typed...)
	case []interface{}:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
	case string:
		values = []string{typed}
	}
	return values
}

func RuleSettingBool(context map[string]interface{}, ruleID, key string, fallback bool) bool {
	cfg, ok := context["config"].(*config.Config)
	if !ok || cfg == nil {
		return fallback
	}
	return cfg.GetRuleSettingBool(ruleID, key, fallback)
}

func IsPathAllowed(context map[string]interface{}, ruleID, path string) bool {
	for _, pattern := range RuleSettingStringSlice(context, ruleID, "allowed_paths") {
		if pattern == path || strings.Contains(path, pattern) {
			return true
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err == nil && matched {
			return true
		}
	}
	return false
}

func MatchesConfiguredName(name string, configured []string) bool {
	for _, candidate := range configured {
		if name == candidate || strings.TrimPrefix(name, ":") == strings.TrimPrefix(candidate, ":") {
			return true
		}
	}
	return false
}
