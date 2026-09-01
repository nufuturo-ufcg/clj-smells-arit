package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

type PrivateMultimethodsRule struct {
	rules.Rule
}

func (r *PrivateMultimethodsRule) Meta() rules.Rule {
	return r.Rule
}

func (r *PrivateMultimethodsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 ||
		!isPrivateContextForm(node) || !r.hasDefMulti(node) {
		return nil
	}

	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID:   r.ID,
		Message:  "Private multimethod detected: defmulti or defmethod declared in a private context (defn-, letfn, or ^:private)",
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}, "A private multimethod may be deliberately internal; the AST does not prove that public extensibility is required.", "visibility-contract", "public-extensibility")
}

func isPrivateContextForm(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol || isShadowedSymbol(head) {
		return false
	}
	return head.Value == "defn-" || head.Value == "letfn" || hasPrivateMetadata(node)
}

func isShadowedSymbol(node *reader.RichNode) bool {
	return node != nil && node.Resolution != nil && node.Resolution.Kind == reader.ResolutionLocal
}

func isUnshadowedMultimethodForm(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol || isShadowedSymbol(head) {
		return false
	}
	// Keep the rule's existing scope: only the special-form spellings are
	// considered here. A qualified alias such as core/defmethod is handled by
	// the language resolver but is not promoted by this rule implicitly.
	if head.Value != "defmulti" && head.Value != "defmethod" {
		return false
	}
	canonical := head.Value
	if head.Resolution != nil && head.Resolution.CanonicalName != "" {
		canonical = head.Resolution.CanonicalName
	}
	if canonical == "defmulti" {
		canonical = "clojure.core/defmulti"
	}
	if canonical == "defmethod" {
		canonical = "clojure.core/defmethod"
	}
	if canonical != "clojure.core/defmulti" && canonical != "clojure.core/defmethod" {
		return false
	}
	known, valid := semantics.CanonicalArityValid(canonical, len(node.Children)-1)
	return known && valid
}

func (r *PrivateMultimethodsRule) hasDefMulti(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if isUnshadowedMultimethodForm(node) {
		return true
	}
	switch node.Type {
	case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
		return false
	}
	defmultis := r.filterNodes(node.Children, r.hasDefMulti)
	return len(defmultis) > 0
}
func (r *PrivateMultimethodsRule) filterNodes(nodes []*reader.RichNode, predicate func(*reader.RichNode) bool) []*reader.RichNode {
	result := []*reader.RichNode{}
	for _, node := range nodes {
		if predicate(node) {
			result = append(result, node)
		}
	}
	return result
}

func hasPrivateMetadata(node *reader.RichNode) bool {
	if node == nil {
		return false
	}

	var checkMeta func(metaNode *reader.RichNode) bool
	checkMeta = func(metaNode *reader.RichNode) bool {
		if metaNode == nil {
			return false
		}
		if metaNode.Type == reader.NodeKeyword && (metaNode.Value == ":private" || metaNode.Value == "private") {
			return true
		}
		if metaNode.Type == reader.NodeMap {
			for i := 0; i+1 < len(metaNode.Children); i += 2 {
				k := metaNode.Children[i]
				if k != nil && k.Type == reader.NodeKeyword && (k.Value == ":private" || k.Value == "private") {
					return true
				}
			}
		}
		if metaNode.Type == reader.NodeMetadata {
			for _, c := range metaNode.Children {
				if checkMeta(c) {
					return true
				}
			}
		}
		return false
	}

	if checkMeta(node.Metadata) {
		return true
	}
	if len(node.Children) > 1 && node.Children[1] != nil &&
		node.Children[1].Type == reader.NodeKeyword && node.Children[1].Value == ":private" {
		return true
	}

	for _, child := range node.Children {
		if child != nil {
			if child.Type == reader.NodeMetadata && checkMeta(child) {
				return true
			}
			if child.Metadata != nil && checkMeta(child.Metadata) {
				return true
			}
		}
	}
	return false
}

func init() {
	defaultRule := &PrivateMultimethodsRule{
		Rule: rules.Rule{
			ID:   "private-multimethods",
			Name: "Private Multimethods",
			Description: "Private multimethod definition: defmulti or defmethod declared in a private context (defn-, letfn, or ^:private). " +
				"Multimethods should remain public to allow open extension.",
			Severity: rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}
