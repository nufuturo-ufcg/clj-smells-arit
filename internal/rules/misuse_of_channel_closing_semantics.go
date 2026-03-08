package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type MisuseOfChannelClosingSemanticsRule struct {
	Rule
}

func (r *MisuseOfChannelClosingSemanticsRule) Meta() Rule {
	return r.Rule
}

func (r *MisuseOfChannelClosingSemanticsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	head := node.Children[0]
	if head.Type != reader.NodeSymbol {
		return nil
	}
	headVal := head.Value

	if isPutSymbol(headVal) {
		for i := 1; i < len(node.Children); i++ {
			child := node.Children[i]
			if child != nil && child.Type == reader.NodeKeyword && isSentinelKeyword(child.Value) {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Using sentinel value %s in put!/>!/>!! to signal stream end. Prefer (close! ch) so that (<! ch) returns nil; avoid custom sentinels.", child.Value),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
		return nil
	}

	if headVal == "not=" || headVal == "=" {
		if len(node.Children) != 3 {
			return nil
		}
		a, b := node.Children[1], node.Children[2]
		sentinel, other := "", (*reader.RichNode)(nil)
		if a != nil && a.Type == reader.NodeKeyword && isSentinelKeyword(a.Value) {
			sentinel, other = a.Value, b
		} else if b != nil && b.Type == reader.NodeKeyword && isSentinelKeyword(b.Value) {
			sentinel, other = b.Value, a
		}
		if sentinel != "" && isChannelReadOrBinding(other) {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Comparison with sentinel %s; if this signals channel termination, prefer closing the channel with close! and use (when-let [e (<! ch)] ...) so nil means closed.", sentinel),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

func isPutSymbol(s string) bool {
	switch s {
	case "put!", ">!", ">!!":
		return true
	}
	return strings.HasSuffix(s, "/put!") || strings.HasSuffix(s, "/>!") || strings.HasSuffix(s, "/>!!")
}

func isTakeSymbol(s string) bool {
	switch s {
	case "<!", "<!!":
		return true
	}
	return strings.HasSuffix(s, "/<!") || strings.HasSuffix(s, "/<!!")
}

func isChannelReadOrBinding(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol {
		return true
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 {
		head := node.Children[0]
		if head != nil && head.Type == reader.NodeSymbol {
			return isTakeSymbol(head.Value)
		}
	}
	return false
}

func isSentinelKeyword(v string) bool {
	switch v {
	case ":done", ":EOF", ":end", "::end", ":eof":
		return true
	}
	if strings.HasSuffix(v, "/end") || strings.HasSuffix(v, "/done") || strings.HasSuffix(v, "/eof") {
		return true
	}
	return false
}

func init() {
	defaultRule := &MisuseOfChannelClosingSemanticsRule{
		Rule: Rule{
			ID:          "misuse-of-channel-closing-semantics",
			Name:        "Misuse of Channel Closing Semantics",
			Description: "Detects using a sentinel value (:done, :EOF, ::end) in put!/>!/>!! or in comparisons to signal channel end; prefer close! and nil from <!.",
			Severity:    SeverityWarning,
		},
	}
	RegisterRule(defaultRule)
}
