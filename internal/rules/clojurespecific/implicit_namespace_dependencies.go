package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/reader"
)

type ImplicitNamespaceDependenciesRule struct {
	rules.Rule
	fileNamespaces map[string]map[string]bool
	fileHasNs      map[string]bool
	mu             sync.Mutex
}

func (r *ImplicitNamespaceDependenciesRule) Meta() rules.Rule {
	return r.Rule
}

func (r *ImplicitNamespaceDependenciesRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if strings.HasSuffix(filepath, "project.clj") {
		return nil
	}
	r.collectNamespaces(node, filepath)

	if node.Type == reader.NodeSymbol {
		return nil
	}

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

func (r *ImplicitNamespaceDependenciesRule) checkUseDirective(node *reader.RichNode, filepath string) *rules.Finding {
	implicitNamespaces := r.extractImplicitNamespacesFromUseDirective(node)
	if len(implicitNamespaces) == 0 {
		return nil
	}

	if isDevOrTestFile(filepath) {
		var filtered []string
		for _, ns := range implicitNamespaces {
			if !isAllowedReferAllNs(ns) {
				filtered = append(filtered, ns)
			}
		}
		implicitNamespaces = filtered
		if len(implicitNamespaces) == 0 {
			return nil
		}
	}

	nsStr := strings.Join(implicitNamespaces, ", ")
	if nsStr == "" {
		nsStr = "unknown"
	}

	return &rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Implicit namespace dependency: :use directive imports all public symbols from [%s]. "+
				"Replace (:use ...) with (:require [ns :refer [specific-symbols]]) or use (:use [ns :only [specific-symbols]]) to list imports explicitly.",
			nsStr,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func (r *ImplicitNamespaceDependenciesRule) checkStandaloneUse(node *reader.RichNode, filepath string) *rules.Finding {
	if r.standaloneUseHasExplicitOnly(node) {
		return nil
	}

	namespaceName := r.extractNameFromStandaloneArg(node)
	if namespaceName == "" {
		namespaceName = "unknown namespace"
	}

	return &rules.Finding{
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

func isDevOrTestFile(filepath string) bool {
	return strings.HasSuffix(filepath, "_test.clj") || strings.Contains(filepath, "/dev/") || strings.Contains(filepath, "/test/") || strings.Contains(filepath, "/int-test/")
}

func isAllowedReferAllNs(nsName string) bool {
	return nsName == "clojure.repl" || nsName == "clojure.test" || nsName == "clojure.tools.namespace.repl" || nsName == "clojure.pprint" || nsName == "alex-and-georges.debug-repl"
}

func (r *ImplicitNamespaceDependenciesRule) checkRequireForReferAll(node *reader.RichNode, filepath string) *rules.Finding {
	var problematicNs []string

	for i := 1; i < len(node.Children); i++ {
		spec := node.Children[i]
		if spec.Type != reader.NodeVector || len(spec.Children) == 0 {
			continue
		}

		if r.vectorContainsReferAll(spec) {
			if spec.Children[0].Type == reader.NodeSymbol {
				nsName := spec.Children[0].Value
				if !(isDevOrTestFile(filepath) && isAllowedReferAllNs(nsName)) {
					problematicNs = append(problematicNs, nsName)
				}
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
				fullNs := subNs
				if prefix != "" && subNs != "" {
					fullNs = prefix + "." + subNs
				}
				if fullNs != "" {
					if !(isDevOrTestFile(filepath) && isAllowedReferAllNs(fullNs)) {
						problematicNs = append(problematicNs, fullNs)
					}
				}
			}
		}
	}

	if len(problematicNs) == 0 {
		return nil
	}

	return &rules.Finding{
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

func (r *ImplicitNamespaceDependenciesRule) extractImplicitNamespacesFromUseDirective(node *reader.RichNode) []string {
	var implicit []string
	for i := 1; i < len(node.Children); i++ {
		child := node.Children[i]
		switch child.Type {
		case reader.NodeSymbol:
			implicit = append(implicit, child.Value)
		case reader.NodeVector:
			if r.useSpecHasExplicitOnly(child) {
				continue
			}
			if len(child.Children) > 0 && child.Children[0].Type == reader.NodeSymbol {
				implicit = append(implicit, child.Children[0].Value)
			} else {
				implicit = append(implicit, "unknown")
			}
		}
	}
	return implicit
}

func (r *ImplicitNamespaceDependenciesRule) useSpecHasExplicitOnly(spec *reader.RichNode) bool {
	if spec == nil || spec.Type != reader.NodeVector {
		return false
	}
	for i := 0; i < len(spec.Children)-1; i++ {
		if spec.Children[i].Type != reader.NodeKeyword || spec.Children[i].Value != ":only" {
			continue
		}
		next := spec.Children[i+1]
		if next.Type == reader.NodeVector || next.Type == reader.NodeList {
			return true
		}
	}
	return false
}

func (r *ImplicitNamespaceDependenciesRule) standaloneUseHasExplicitOnly(node *reader.RichNode) bool {
	for i := 1; i < len(node.Children); i++ {
		child := node.Children[i]
		if child.Type != reader.NodeQuote || len(child.Children) == 0 {
			continue
		}
		inner := child.Children[0]
		if inner.Type == reader.NodeVector && r.useSpecHasExplicitOnly(inner) {
			return true
		}
	}
	return false
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

func isCommonOrCoreNamespace(prefix string) bool {
	// clojure.core e cljs.core são sempre disponíveis sem :require
	if prefix == "clojure.core" || prefix == "cljs.core" {
		return true
	}
	// java.lang é importado automaticamente pela JVM
	if strings.HasPrefix(prefix, "java.lang.") || prefix == "java.lang" {
		return true
	}
	// ClojureScript host/browser namespaces — sempre globais, nunca declaradas em :require
	switch prefix {
	case "js", "goog", "Math", "console", "window", "document", "navigator",
		"location", "history", "XMLHttpRequest", "Promise", "Error",
		"Object", "Array", "JSON", "Date", "RegExp", "String", "Number",
		"Boolean", "Symbol", "Map", "Set", "WeakMap", "WeakSet",
		"setTimeout", "clearTimeout", "setInterval", "clearInterval",
		"requestAnimationFrame", "cancelAnimationFrame",
		"fetch", "Headers", "Request", "Response",
		"localStorage", "sessionStorage", "indexedDB",
		"performance", "crypto":
		return true
	}
	return false
}

func (r *ImplicitNamespaceDependenciesRule) collectNamespaces(node *reader.RichNode, filepath string) {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return
	}
	first := node.Children[0]
	if first.Type != reader.NodeSymbol {
		return
	}

	r.mu.Lock()
	if r.fileNamespaces == nil {
		r.fileNamespaces = make(map[string]map[string]bool)
		r.fileHasNs = make(map[string]bool)
	}
	if r.fileNamespaces[filepath] == nil {
		r.fileNamespaces[filepath] = make(map[string]bool)
	}
	allowed := r.fileNamespaces[filepath]
	r.mu.Unlock()

	if first.Value == "ns" {
		r.mu.Lock()
		r.fileHasNs[filepath] = true
		r.mu.Unlock()
		for i := 1; i < len(node.Children); i++ {
			child := node.Children[i]
			if child.Type == reader.NodeList && len(child.Children) > 0 {
				kwd := child.Children[0].Value
				if kwd == ":require" || kwd == ":use" || kwd == "require" || kwd == "use" {
					r.extractNamespacesFromArgs(child, allowed)
				}
			}
		}
	} else if first.Value == "require" || first.Value == "use" {
		r.extractNamespacesFromArgs(node, allowed)
	}
}

func (r *ImplicitNamespaceDependenciesRule) extractNamespacesFromArgs(reqNode *reader.RichNode, allowed map[string]bool) {
	for i := 1; i < len(reqNode.Children); i++ {
		child := reqNode.Children[i]
		if child.Type == reader.NodeSymbol {
			allowed[child.Value] = true
		} else if child.Type == reader.NodeQuote && len(child.Children) > 0 {
			r.extractNamespacesFromArgs(&reader.RichNode{Children: append([]*reader.RichNode{nil}, child.Children[0])}, allowed)
		} else if child.Type == reader.NodeVector && len(child.Children) > 0 {
			if child.Children[0].Type == reader.NodeSymbol {
				nsName := child.Children[0].Value
				allowed[nsName] = true
				for j := 1; j < len(child.Children)-1; j++ {
					if child.Children[j].Type == reader.NodeKeyword && child.Children[j].Value == ":as" {
						if child.Children[j+1].Type == reader.NodeSymbol {
							allowed[child.Children[j+1].Value] = true
						}
					}
				}
			}
		} else if child.Type == reader.NodeList && len(child.Children) > 0 {
			if child.Children[0].Type == reader.NodeSymbol {
				prefix := child.Children[0].Value
				for j := 1; j < len(child.Children); j++ {
					sub := child.Children[j]
					if sub.Type == reader.NodeSymbol {
						allowed[prefix+"."+sub.Value] = true
					} else if sub.Type == reader.NodeVector && len(sub.Children) > 0 && sub.Children[0].Type == reader.NodeSymbol {
						subNs := sub.Children[0].Value
						allowed[prefix+"."+subNs] = true
						for k := 1; k < len(sub.Children)-1; k++ {
							if sub.Children[k].Type == reader.NodeKeyword && sub.Children[k].Value == ":as" {
								if sub.Children[k+1].Type == reader.NodeSymbol {
									allowed[sub.Children[k+1].Value] = true
								}
							}
						}
					}
				}
			}
		}
	}
}

func init() {
	defaultRule := &ImplicitNamespaceDependenciesRule{
		Rule: rules.Rule{
			ID:          "implicit-namespace-dependencies",
			Name:        "Implicit Namespace Dependencies",
			Description: "Detects implicit namespace dependencies introduced by :use without :only, :refer :all in :require, or standalone (use ...) without :only. :use [ns :only [syms]] lists explicit imports and is not reported. Unrestricted imports cause symbol ambiguity, namespace pollution, and dependencies that static analysis tools cannot reliably resolve.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}