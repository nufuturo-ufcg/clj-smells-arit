package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type ImplicitNamespaceDependenciesRule struct {
	Rule
}

func (r *ImplicitNamespaceDependenciesRule) Meta() Rule {
	return r.Rule
}

func (r *ImplicitNamespaceDependenciesRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	first := node.Children[0]

	if first.Type == reader.NodeKeyword && first.Value == ":use" {
		return r.checkUseDirective(node, filepath)
	}

	if first.Type == reader.NodeSymbol && first.Value == "use" {
		return r.checkStandaloneUse(node, filepath)
	}

	if first.Type == reader.NodeKeyword && first.Value == ":require" {
		return r.checkRequireForReferAll(node, filepath)
	}

	return nil
}

func (r *ImplicitNamespaceDependenciesRule) checkUseDirective(node *reader.RichNode, filepath string) *Finding {
	namespaces := r.extractNamespacesFromDirective(node)
	nsStr := strings.Join(namespaces, ", ")
	if nsStr == "" {
		nsStr = "unknown"
	}

	return &Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Implicit namespace dependency: :use directive imports all public symbols from [%s]. "+
				"Replace (:use ...) with (:require [ns :refer [specific-symbols]]) to make dependencies explicit.",
			nsStr,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func (r *ImplicitNamespaceDependenciesRule) checkStandaloneUse(node *reader.RichNode, filepath string) *Finding {
	namespaceName := r.extractNameFromStandaloneArg(node)
	if namespaceName == "" {
		namespaceName = "unknown namespace"
	}

	return &Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Implicit namespace dependency: standalone (use '%s) imports all public symbols. "+
				"Replace with (require '[%s :refer [specific-symbols]]) or use the ns :require form.",
			namespaceName, namespaceName,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func (r *ImplicitNamespaceDependenciesRule) checkRequireForReferAll(node *reader.RichNode, filepath string) *Finding {
	var problematicNs []string

	for i := 1; i < len(node.Children); i++ {
		spec := node.Children[i]
		if spec.Type != reader.NodeVector || len(spec.Children) == 0 {
			continue
		}

		if r.vectorContainsReferAll(spec) {
			if spec.Children[0].Type == reader.NodeSymbol {
				problematicNs = append(problematicNs, spec.Children[0].Value)
			}
		}

		for _, child := range spec.Children {
			if child.Type == reader.NodeVector && r.vectorContainsReferAll(child) {
				prefix := ""
				if spec.Children[0].Type == reader.NodeSymbol {
					prefix = spec.Children[0].Value
				}
				subNs := ""
				if len(child.Children) > 0 && child.Children[0].Type == reader.NodeSymbol {
					subNs = child.Children[0].Value
				}
				if prefix != "" && subNs != "" {
					problematicNs = append(problematicNs, prefix+"."+subNs)
				} else if subNs != "" {
					problematicNs = append(problematicNs, subNs)
				}
			}
		}
	}

	if len(problematicNs) == 0 {
		return nil
	}

	return &Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Implicit namespace dependency: :refer :all in :require for [%s] imports all public symbols. "+
				"Use explicit :refer [specific-symbols] to make dependencies clear and avoid name collisions.",
			strings.Join(problematicNs, ", "),
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func (r *ImplicitNamespaceDependenciesRule) vectorContainsReferAll(v *reader.RichNode) bool {
	for i := 0; i < len(v.Children)-1; i++ {
		if v.Children[i].Type == reader.NodeKeyword && v.Children[i].Value == ":refer" &&
			v.Children[i+1].Type == reader.NodeKeyword && v.Children[i+1].Value == ":all" {
			return true
		}
	}
	return false
}

func (r *ImplicitNamespaceDependenciesRule) extractNamespacesFromDirective(node *reader.RichNode) []string {
	var namespaces []string
	for i := 1; i < len(node.Children); i++ {
		child := node.Children[i]
		switch child.Type {
		case reader.NodeSymbol:
			namespaces = append(namespaces, child.Value)
		case reader.NodeVector:
			if len(child.Children) > 0 && child.Children[0].Type == reader.NodeSymbol {
				namespaces = append(namespaces, child.Children[0].Value)
			}
		}
	}
	return namespaces
}

func (r *ImplicitNamespaceDependenciesRule) extractNameFromStandaloneArg(node *reader.RichNode) string {
	for i := 1; i < len(node.Children); i++ {
		child := node.Children[i]
		switch child.Type {
		case reader.NodeSymbol:
			return child.Value
		case reader.NodeQuote:
			if len(child.Children) > 0 {
				inner := child.Children[0]
				if inner.Type == reader.NodeSymbol {
					return inner.Value
				}
				if inner.Type == reader.NodeVector && len(inner.Children) > 0 && inner.Children[0].Type == reader.NodeSymbol {
					return inner.Children[0].Value
				}
			}
		}
	}
	return ""
}

func init() {
	defaultRule := &ImplicitNamespaceDependenciesRule{
		Rule: Rule{
			ID:          "implicit-namespace-dependencies",
			Name:        "Implicit Namespace Dependencies",
			Description: "Detects implicit namespace dependencies introduced by :use directives, :refer :all in :require, or standalone (use ...) calls. These patterns import all symbols from a namespace without explicitly listing them, causing symbol ambiguity, namespace pollution, and implicit dependencies that static analysis tools cannot reliably resolve.",
			Severity:    SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
