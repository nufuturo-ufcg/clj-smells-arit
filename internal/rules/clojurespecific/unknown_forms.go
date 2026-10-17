package clojurespecific

import (
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// argumentOfUnknownForm reports whether the node being checked is a direct
// argument of a call whose head may be a macro that never evaluates it, e.g.
// labrepl's (code (def x 1)), which quotes its arguments for display. The head
// is known to evaluate its arguments only when it resolves to clojure.core, a
// local binding, a Java member, or a function defined with defn in this file.
// Every other head (a var of another namespace, an unresolved symbol, or a
// macro of this file) is treated as unknown, so the node is not proven to run.
func argumentOfUnknownForm(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) == 0 {
		return false
	}
	parent := ancestors[len(ancestors)-1]
	if parent == nil || parent.Type != reader.NodeList || len(parent.Children) == 0 {
		return false
	}
	head := parent.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return false
	}
	// (letfn [(f [x] body)] ...): the parent is a function spec, not a call.
	if len(ancestors) >= 3 && ancestors[len(ancestors)-2] != nil && ancestors[len(ancestors)-2].Type == reader.NodeVector {
		if letfn := ancestors[len(ancestors)-3]; letfn != nil && letfn.Type == reader.NodeList && len(letfn.Children) > 1 &&
			letfn.Children[1] == ancestors[len(ancestors)-2] && letfn.Children[0] != nil &&
			strings.TrimPrefix(letfn.Children[0].Value, "clojure.core/") == "letfn" {
			return false
		}
	}
	if resolution := head.Resolution; resolution != nil {
		switch resolution.Kind {
		case reader.ResolutionClojureCore,
			reader.ResolutionJavaStatic, reader.ResolutionJavaConstructor, reader.ResolutionJavaMethod:
			return false
		case reader.ResolutionLocal:
			// The resolver also marks vars referred from another namespace
			// as local; only a binding introduced by an enclosing form is a
			// known local function.
			if head.ResolvedDefinition != nil {
				for _, ancestor := range ancestors {
					if nodeWithin(ancestor, head.ResolvedDefinition) {
						return false
					}
				}
			}
		}
	}
	name := strings.TrimPrefix(head.Value, "clojure.core/")
	if verboseEvaluatingCoreForms[name] || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}
	if strings.Contains(name, "/") {
		return true
	}
	if _, macro := fileMacros(context)[name]; macro {
		return true
	}
	return !fileDefinesFunction(context, name)
}

// fileDefinesFunction reports whether the file defines name with defn or defn-.
func fileDefinesFunction(context map[string]interface{}, name string) bool {
	roots, _ := context["file-roots"].([]*reader.RichNode)
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) < 2 || root.Children[0] == nil {
			continue
		}
		switch root.Children[0].Value {
		case "defn", "defn-":
			if macroDeclaredName(root) == name {
				return true
			}
		}
	}
	return false
}
