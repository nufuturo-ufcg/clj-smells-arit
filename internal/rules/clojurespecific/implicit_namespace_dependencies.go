package clojurespecific

import (
	"fmt"
	"github.com/thlaurentino/arit/internal/rules"
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
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
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

	// Dependency clauses exist only directly inside ns; elsewhere (:use m)
	// is a keyword lookup. The ns macro also accepts the colon-less spelling.
	inNs := implicitParentIsNs(context)
	if inNs && (first.Type == reader.NodeKeyword || first.Type == reader.NodeSymbol) && strings.TrimPrefix(first.Value, ":") == "use" {
		return r.checkUseDirective(node, context, filepath)
	}

	if !inNs && first.Type == reader.NodeSymbol && first.Value == "use" {
		return r.checkStandaloneUse(node, context, filepath)
	}

	if inNs && (first.Type == reader.NodeKeyword || first.Type == reader.NodeSymbol) && strings.TrimPrefix(first.Value, ":") == "require" {
		return r.checkRequireForReferAll(node, context, filepath)
	}

	if first.Type == reader.NodeSymbol && strings.Contains(first.Value, "/") && rules.ProjectSemanticIndex(context) != nil {
		return r.checkQualifiedDependency(node, context, filepath)
	}

	return nil
}

func implicitParentIsNs(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) == 0 {
		return false
	}
	// #?(:clj (:use ...)) and #?@(:clj [(:use ...)]) are clauses of the
	// enclosing ns.
	index := len(ancestors) - 1
	isReaderCond := func(node *reader.RichNode) bool {
		return node != nil && (node.Type == reader.NodeReaderCond || node.Type == reader.NodeReaderCondSplice)
	}
	for index > 0 && (isReaderCond(ancestors[index]) ||
		(ancestors[index] != nil && ancestors[index].Type == reader.NodeVector && isReaderCond(ancestors[index-1]))) {
		index--
	}
	parent := ancestors[index]
	return parent != nil && parent.Type == reader.NodeList && len(parent.Children) > 0 &&
		parent.Children[0].Type == reader.NodeSymbol &&
		(parent.Children[0].Value == "ns" || parent.Children[0].Value == "clojure.core/ns")
}

func (r *ImplicitNamespaceDependenciesRule) checkQualifiedDependency(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	name := node.Children[0].Value
	separator := strings.Index(name, "/")
	if separator <= 0 {
		return nil
	}
	namespace := name[:separator]
	if isCommonOrCoreNamespace(namespace) || namespace == rules.CurrentNamespace(context) {
		return nil
	}
	if resolution := node.Children[0].Resolution; resolution != nil {
		switch resolution.Kind {
		case reader.ResolutionJavaStatic, reader.ResolutionJavaConstructor, reader.ResolutionJavaMethod:
			return nil
		}
	}
	required, _ := context["namespace-requires"].(map[string]bool)
	aliases, _ := context["namespace-aliases"].(map[string]string)
	if required[namespace] || aliases[namespace] != "" {
		return nil
	}
	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID:   r.ID,
		Message:  fmt.Sprintf("Qualified namespace dependency `%s` is used without a visible local :require declaration. Declare the dependency explicitly or document the project-level loading contract.", name),
		Filepath: filepath,
		Location: node.Children[0].Location,
		Severity: rules.ContextualSeverity(context, r.Severity),
		Tags:     rules.ContextualTags(context),
	}, "A qualified symbol may be provided by a project-level loader or a deliberate runtime dependency.", "namespace-loading-contract", "project-dependency-index")
}

func (r *ImplicitNamespaceDependenciesRule) checkUseDirective(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	implicitNamespaces := r.extractImplicitNamespacesFromUseDirective(node)
	if len(implicitNamespaces) == 0 {
		return nil
	}

	nsStr := strings.Join(implicitNamespaces, ", ")
	if nsStr == "" {
		nsStr = "unknown"
	}

	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Implicit namespace dependency: :use directive imports all public symbols from [%s]. "+
				"Replace (:use ...) with (:require [ns :refer [specific-symbols]]) or use (:use [ns :only [specific-symbols]]) to list imports explicitly.",
			nsStr,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: rules.ContextualSeverity(context, r.Severity),
		Tags:     rules.ContextualTags(context),
	}, "Broad imports may be deliberate in REPLs, scripts, or DSLs; the AST does not prove a collision or an unsuitable contract.", "namespace-import-scope", "symbol-collision-contract")
}

func (r *ImplicitNamespaceDependenciesRule) checkStandaloneUse(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if isInteropMethodCall(context) || argumentOfUnknownForm(context) {
		return nil
	}
	// Only quoted arguments are known at analysis time; an evaluated argument
	// may carry :only.
	var implicit []string
	for _, child := range node.Children[1:] {
		if child.Type == reader.NodeQuote && len(child.Children) > 0 {
			implicit = append(implicit, r.implicitNamespacesOfUseSpec(child.Children[0], "")...)
		}
	}
	if len(implicit) == 0 {
		return nil
	}
	namespaceName := strings.Join(implicit, ", ")

	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Implicit namespace dependency: standalone (use '%s) imports all public symbols. "+
				"Replace with (require '[%s :refer [specific-symbols]]) or use the ns :require form.",
			namespaceName, namespaceName,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: rules.ContextualSeverity(context, r.Severity),
		Tags:     rules.ContextualTags(context),
	}, "Broad imports may be deliberate in REPLs, scripts, or DSLs; the AST does not prove a collision or an unsuitable contract.", "namespace-import-scope", "symbol-collision-contract")
}

func isInteropMethodCall(context map[string]interface{}) bool {
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil || parent.Type != reader.NodeList || len(parent.Children) == 0 {
		return false
	}
	head := parent.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return false
	}
	return head.Value == "." || head.Value == ".."
}

func (r *ImplicitNamespaceDependenciesRule) checkRequireForReferAll(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	var problematicNs []string

	for i := 1; i < len(node.Children); i++ {
		spec := node.Children[i]
		if spec.Type != reader.NodeVector || len(spec.Children) == 0 {
			continue
		}

		if r.vectorContainsReferAll(spec) {
			if spec.Children[0].Type == reader.NodeSymbol {
				nsName := spec.Children[0].Value
				problematicNs = append(problematicNs, nsName)
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
					problematicNs = append(problematicNs, fullNs)
				}
			}
		}
	}

	if len(problematicNs) == 0 {
		return nil
	}

	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Implicit namespace dependency: :refer :all in :require for [%s] imports all public symbols. "+
				"Use explicit :refer [specific-symbols] to make dependencies clear and avoid name collisions.",
			strings.Join(problematicNs, ", "),
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: rules.ContextualSeverity(context, r.Severity),
		Tags:     rules.ContextualTags(context),
	}, "Using :refer :all may be deliberate in internal APIs, REPLs, or DSLs; the AST does not prove a collision or an unsuitable contract.", "namespace-import-scope", "symbol-collision-contract")
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
		implicit = append(implicit, r.implicitNamespacesOfUseSpec(node.Children[i], "")...)
	}
	return implicit
}

// implicitNamespacesOfUseSpec returns the namespaces a use argument imports
// without :only. An argument is a lib symbol, a libspec [lib & options], or a
// prefix list [prefix lib-or-libspec ...] whose entries name prefix.lib.
func (r *ImplicitNamespaceDependenciesRule) implicitNamespacesOfUseSpec(spec *reader.RichNode, prefix string) []string {
	qualify := func(name string) string {
		if prefix == "" {
			return name
		}
		return prefix + "." + name
	}
	switch spec.Type {
	case reader.NodeSymbol:
		return []string{qualify(spec.Value)}
	case reader.NodeVector, reader.NodeList:
		if len(spec.Children) == 0 || spec.Children[0].Type != reader.NodeSymbol {
			return nil
		}
		if prefix == "" && len(spec.Children) > 1 &&
			(spec.Children[1].Type == reader.NodeSymbol || spec.Children[1].Type == reader.NodeVector || spec.Children[1].Type == reader.NodeList) {
			var implicit []string
			for _, entry := range spec.Children[1:] {
				implicit = append(implicit, r.implicitNamespacesOfUseSpec(entry, spec.Children[0].Value)...)
			}
			return implicit
		}
		if r.useSpecHasExplicitOnly(spec) {
			return nil
		}
		return []string{qualify(spec.Children[0].Value)}
	}
	return nil
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

func isCommonOrCoreNamespace(prefix string) bool {
	// clojure.core and cljs.core are always available without :require
	if prefix == "clojure.core" || prefix == "cljs.core" {
		return true
	}
	// java.lang is imported automatically by the JVM
	if strings.HasPrefix(prefix, "java.lang.") || prefix == "java.lang" {
		return true
	}
	// ClojureScript host/browser namespaces — always global and never declared in :require
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
			ID:                    "implicit-namespace-dependencies",
			Name:                  "Implicit Namespace Dependencies",
			Description:           "Detects implicit namespace dependencies introduced by :use without :only, :refer :all in :require, or standalone (use ...) without :only. :use [ns :only [syms]] lists explicit imports and is not reported. Unrestricted imports cause symbol ambiguity, namespace pollution, and dependencies that static analysis tools cannot reliably resolve.",
			ContextualDescription: "It may be contextual in REPLs, scripts, DSLs, and development code; in those cases, the finding is marked contextual and appears with --include-contextual.",
			Severity:              rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}
