package rules

import (
	"github.com/thlaurentino/arit/internal/reader"
)

type UseMapcatRule struct {
}

func (r *UseMapcatRule) Meta() Rule {
	return Rule{
		ID:          "underutilizing-features: use-mapcat",
		Name:        "Underutilizing features: Use mapcat",
		Description: "Detects usage of (apply concat (map ...)) which can be replaced by (mapcat ...)",
		Severity:    SeverityHint,
	}
}

func (r *UseMapcatRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList {
		return nil
	}

	if len(node.Children) != 3 {
		return nil
	}

	applyNode := node.Children[0]
	concatNode := node.Children[1]
	mapFormNode := node.Children[2]

	if applyNode.Type != reader.NodeSymbol || applyNode.Value != "apply" {
		return nil
	}

	if concatNode.Type != reader.NodeSymbol || concatNode.Value != "concat" {
		return nil
	}

	if mapFormNode.Type != reader.NodeList {
		return nil
	}

	if len(mapFormNode.Children) < 2 || mapFormNode.Children[0].Type != reader.NodeSymbol || mapFormNode.Children[0].Value != "map" {
		return nil
	}

	meta := r.Meta()
	finding := &Finding{
		RuleID:   meta.ID,
		Message:  "Consider using `mapcat` instead of `(apply concat (map ...))` for better performance and idiomatic style.",
		Filepath: filepath,
		Location: node.Location,
		Severity: meta.Severity,
	}
	return finding
}

func init() {

	defaultRule := &UseMapcatRule{}

	RegisterRule(defaultRule)
}
