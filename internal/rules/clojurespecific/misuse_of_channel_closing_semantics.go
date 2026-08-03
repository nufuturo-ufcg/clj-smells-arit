package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type MisuseOfChannelClosingSemanticsRule struct {
	rules.Rule
}

func (r *MisuseOfChannelClosingSemanticsRule) Meta() rules.Rule {
	return r.Rule
}

func (r *MisuseOfChannelClosingSemanticsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	head := node.Children[0]
	if head.Type == reader.NodeKeyword {
		if isSentinelKeyword(head.Value) && isInsideGoBlock(context) {
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Checking sentinel %s: prefer closing channels and checking for nil.", head.Value),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
		return nil
	}

	if head.Type != reader.NodeSymbol {
		return nil
	}
	headVal := head.Value

	if isPutSymbol(headVal) {
		if len(node.Children) >= 3 {
			valueArg := node.Children[2]
			sentinel := findSentinelInNode(valueArg)
			if sentinel != "" {
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Sentinel value %s in %s: prefer (close! ch) so that (<! ch) returns nil; avoid custom sentinels.", sentinel, headVal),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
		return nil
	}

	if (headVal == "not=" || headVal == "=") && isInsideGoBlock(context) {
		var sentinel string
		for _, child := range node.Children[1:] {
			if s := findSentinelInNode(child); s != "" {
				sentinel = s
				break
			}
			if child.Type == reader.NodeNumber && child.Value == "-1" {
				sentinel = "-1"
				break
			}
		}
		if sentinel != "" {
			return &rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Comparison with sentinel %s: prefer (close! ch) so that (<! ch) returns nil; use (when-let [e (<! ch)] ...) when closed.", sentinel),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	if (headVal == "contains?" || headVal == "get") && isInsideGoBlock(context) {
		if len(node.Children) >= 3 {
			keyArg := node.Children[2]
			if sentinel := findSentinelInNode(keyArg); sentinel != "" {
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Checking sentinel key %s: prefer closing channels and checking for nil.", sentinel),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	return nil
}

func isInsideGoBlock(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	for _, form := range enclosing {
		if form == "go" || form == "go-loop" || strings.HasSuffix(form, "/go") || strings.HasSuffix(form, "/go-loop") {
			return true
		}
	}
	return false
}

func findSentinelInNode(node *reader.RichNode) string {
	if node == nil {
		return ""
	}
	if node.Type == reader.NodeKeyword || node.Type == reader.NodeSymbol || node.Type == reader.NodeString {
		if isSentinelKeyword(node.Value) {
			return node.Value
		}
	}
	for _, child := range node.Children {
		if res := findSentinelInNode(child); res != "" {
			return res
		}
	}
	return ""
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
	"synced", "return", "break", "nil", "last-item", "shutdown",
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
		Rule: rules.Rule{
			ID:          "misuse-of-channel-closing-semantics",
			Name:        "Misuse of Channel Closing Semantics",
			Description: "Flags keywords that look like stream-end sentinels (done, end, close, stop, complete, etc., word-boundary) in put!/>!/>!! or in comparisons with <!/<!!. Avoids test/placeholder values (:foo, :test-val). Prefer close! and nil from <!.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}