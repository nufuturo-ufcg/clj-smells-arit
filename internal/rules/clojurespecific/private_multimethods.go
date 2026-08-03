package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type PrivateMultimethodsRule struct {
	rules.Rule
}

func (r *PrivateMultimethodsRule) Meta() rules.Rule {
	return r.Rule
}

func (r *PrivateMultimethodsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node.Type == reader.NodeList && len(node.Children) > 1 {
		if node.Children[0].Type == reader.NodeSymbol {
			if node.Children[0].Value == "defn-" || node.Children[0].Value == "letfn" || r.hasPrivateMetadata(node) {
				if r.hasDefMulti(node) {
					return &rules.Finding{
						RuleID:   r.ID,
						Message:  fmt.Sprintf("Private multimethod detected: defmulti or defmethod declared"+
						"in a private context (defn-, letfn, or ^:private)"),
						Filepath: filepath,
						Location: node.Location,
						Severity: r.Severity,
					}
				}
			}
		}
	}
	return nil
}


func (r *PrivateMultimethodsRule) hasDefMulti(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if r.isDefMulti(node) {
		return true
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

func (r *PrivateMultimethodsRule) isDefMulti(node *reader.RichNode) bool {
	return node.Type == reader.NodeList &&
		len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defmulti" || node.Children[0].Value == "defmethod")
}

func (r *PrivateMultimethodsRule) hasPrivateMetadata(node *reader.RichNode) bool {
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

	for _, child := range node.Children {
		if child != nil {
			if child.Type == reader.NodeKeyword && (child.Value == ":private" || child.Value == "private" || child.Value == "private true") {
				return true
			}
			if checkMeta(child) {
				return true
			}
			if checkMeta(child.Metadata) {
				return true
			}
		}
	}
	return false
}

func init() {
	defaultRule := &PrivateMultimethodsRule{
		Rule: rules.Rule{
			ID:          "private-multimethods",
			Name:        "Private Multimethods",
			Description: "Private multimethod definition: defmulti or defmethod declared in a private context (defn-, letfn, or ^:private). "+
             "Multimethods should remain public to allow open extension.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}