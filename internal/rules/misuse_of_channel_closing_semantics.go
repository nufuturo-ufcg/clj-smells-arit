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
		if len(node.Children) >= 3 {
			valueArg := node.Children[2]
			if valueArg != nil && valueArg.Type == reader.NodeKeyword && isSentinelKeyword(valueArg.Value) {
				return &Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Using sentinel value %s in %s to signal stream end. Prefer (close! ch) so that (<! ch) returns nil; avoid custom sentinels.", valueArg.Value, headVal),
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
		if sentinel != "" && isChannelTakeForm(other) {
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

func isChannelTakeForm(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return false
	}
	return isTakeSymbol(head.Value)
}

var sentinelStems = []string{
	"done", "end", "eof", "close", "stop", "exit",
	"complete", "finish", "eos", "poison", "bye", "quit", "terminat",
	"closed", "finished", "completed",
	"synced", "return", "break", 
}

func isSentinelKeyword(v string) bool {
	local := keywordLocalName(v)
	if local == "" {
		return false
	}
	lower := strings.ToLower(local)
	for _, stem := range sentinelStems {
		if strings.Contains(lower, stem) {
			return true
		}
	}
	return false
}

func keywordLocalName(kw string) string {
	s := kw
	for strings.HasPrefix(s, ":") {
		s = s[1:]
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func init() {
	defaultRule := &MisuseOfChannelClosingSemanticsRule{
		Rule: Rule{
			ID:          "misuse-of-channel-closing-semantics",
			Name:        "Misuse of Channel Closing Semantics",
			Description: "Flags keywords that look like stream-end sentinels (done, end, close, stop, complete, etc., word-boundary) in put!/>!/>!! or in comparisons with <!/<!!. Avoids test/placeholder values (:foo, :test-val). Prefer close! and nil from <!.",
			Severity:    SeverityWarning,
		},
	}
	RegisterRule(defaultRule)
}
