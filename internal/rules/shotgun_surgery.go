package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
)

type ShotgunSurgeryRule struct {
	Rule
	maxExternalDependencies int
	maxNamespaceUsage       int
}

func NewShotgunSurgeryRule(cfg *config.Config) *ShotgunSurgeryRule {
	rule := &ShotgunSurgeryRule{
		Rule: Rule{
			ID:          "shotgun-surgery",
			Name:        "Shotgun Surgery",
			Description: "Detects functions that use many external dependencies, indicating that changes to this function might require changes in many other places. This violates the principle of localized changes and can make maintenance difficult.",
			Severity:    SeverityWarning,
		},
		maxExternalDependencies: 4,
		maxNamespaceUsage:       6,
	}

	if cfg != nil {
		rule.maxExternalDependencies = cfg.GetRuleSettingInt("shotgun-surgery", "max-external-dependencies", rule.maxExternalDependencies)
		rule.maxNamespaceUsage = cfg.GetRuleSettingInt("shotgun-surgery", "max-namespace-usage", rule.maxNamespaceUsage)
	}

	return rule
}

type DependencyAnalysis struct {
	externalNamespaces map[string]int
	totalExternalCalls int
	functionName       string
}

func (r *ShotgunSurgeryRule) Meta() Rule {
	return r.Rule
}

func (r *ShotgunSurgeryRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if !r.isFunction(node) {
		return nil
	}

	analysis := r.analyzeDependencies(node, context, filepath)

	externalNamespaceCount := len(analysis.externalNamespaces)

	if externalNamespaceCount > r.maxExternalDependencies || analysis.totalExternalCalls > r.maxNamespaceUsage {
		namespaces := r.getNamespaceList(analysis.externalNamespaces)

		return &Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf("Function '%s' uses %d external namespaces (%s) with %d total external calls. This indicates potential shotgun surgery - changes to this function might require changes in many other places. Consider breaking down the function or reducing external dependencies.",
				analysis.functionName, externalNamespaceCount, strings.Join(namespaces, ", "), analysis.totalExternalCalls),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *ShotgunSurgeryRule) isFunction(node *reader.RichNode) bool {
	return node.Type == reader.NodeList &&
		len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-")
}

func (r *ShotgunSurgeryRule) analyzeDependencies(node *reader.RichNode, context map[string]interface{}, filepath string) *DependencyAnalysis {
	analysis := &DependencyAnalysis{
		externalNamespaces: make(map[string]int),
		functionName:       r.getFunctionName(node),
	}

	currentNamespace := r.getCurrentNamespace(context, filepath)

	aliases := r.getNamespaceAliases(context)

	r.visitNode(node, analysis, currentNamespace, aliases)

	return analysis
}

func (r *ShotgunSurgeryRule) visitNode(node *reader.RichNode, analysis *DependencyAnalysis, currentNamespace string, aliases map[string]string) {
	if node == nil {
		return
	}

	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		funcCall := node.Children[0].Value

		if strings.Contains(funcCall, "/") {
			parts := strings.Split(funcCall, "/")
			if len(parts) == 2 {
				namespaceOrAlias := parts[0]

				actualNamespace := namespaceOrAlias
				if fullNamespace, exists := aliases[namespaceOrAlias]; exists {
					actualNamespace = fullNamespace
				}

				if actualNamespace != currentNamespace && !r.isClojureCore(actualNamespace) {
					analysis.externalNamespaces[actualNamespace]++
					analysis.totalExternalCalls++
				}
			}
		}
	}

	for _, child := range node.Children {
		r.visitNode(child, analysis, currentNamespace, aliases)
	}
}

func (r *ShotgunSurgeryRule) getCurrentNamespace(context map[string]interface{}, filepath string) string {

	if ns, ok := context["current-namespace"].(string); ok && ns != "" {
		return ns
	}

	if strings.Contains(filepath, "/") {
		parts := strings.Split(filepath, "/")
		if len(parts) > 1 {

			lastPart := parts[len(parts)-1]
			if strings.HasSuffix(lastPart, ".clj") {
				lastPart = strings.TrimSuffix(lastPart, ".clj")
			}

			namespaceParts := append(parts[1:len(parts)-1], lastPart)
			return strings.Join(namespaceParts, ".")
		}
	}

	return "unknown"
}

func (r *ShotgunSurgeryRule) getNamespaceAliases(context map[string]interface{}) map[string]string {
	aliases := make(map[string]string)

	if aliasMap, ok := context["namespace-aliases"].(map[string]string); ok {
		return aliasMap
	}

	aliases["str"] = "clojure.string"
	aliases["set"] = "clojure.set"
	aliases["walk"] = "clojure.walk"
	aliases["data"] = "clojure.data"
	aliases["io"] = "clojure.java.io"
	aliases["json"] = "clojure.data.json"
	aliases["csv"] = "clojure.data.csv"
	aliases["xml"] = "clojure.data.xml"
	aliases["zip"] = "clojure.zip"
	aliases["test"] = "clojure.test"
	aliases["spec"] = "clojure.spec.alpha"
	aliases["s"] = "clojure.spec.alpha"
	aliases["gen"] = "clojure.spec.gen.alpha"
	aliases["edn"] = "clojure.edn"

	return aliases
}

func (r *ShotgunSurgeryRule) isClojureCore(namespace string) bool {

	coreNamespaces := map[string]bool{
		"clojure.core": true,
		"clojure.main": true,
		"clojure.repl": true,
	}

	return coreNamespaces[namespace]
}

func (r *ShotgunSurgeryRule) getFunctionName(node *reader.RichNode) string {
	if node.Type == reader.NodeList && len(node.Children) > 1 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-") {
		if node.Children[1].Type == reader.NodeSymbol {
			return node.Children[1].Value
		}
	}
	return "<unknown_function>"
}

func (r *ShotgunSurgeryRule) getNamespaceList(namespaces map[string]int) []string {
	var result []string
	for ns := range namespaces {
		result = append(result, ns)
	}
	return result
}

func init() {
	RegisterRule(NewShotgunSurgeryRule(nil))
}
