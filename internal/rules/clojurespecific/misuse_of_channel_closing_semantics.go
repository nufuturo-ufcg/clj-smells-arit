package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

type MisuseOfChannelClosingSemanticsRule struct {
	rules.Rule
}

func (r *MisuseOfChannelClosingSemanticsRule) Meta() rules.Rule {
	return r.Rule
}

func (r *MisuseOfChannelClosingSemanticsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}
	if isInsideProtocolDeclaration(context) {
		return nil
	}

	head := node.Children[0]
	if head.Type == reader.NodeKeyword {
		if isSentinelKeyword(head.Value) && isInsideGoBlock(context) {
			return rules.SetContextualFindingWithEvidence(&rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Checking sentinel %s: prefer closing channels and checking for nil.", head.Value),
				Filepath: filepath,
				Location: node.Location,
				Severity: rules.ContextualSeverity(context, r.Severity),
				Tags:     rules.ContextualTags(context),
			}, "A sentinel may be a legitimate part of the channel's domain protocol.", "channel-domain-protocol", "close-semantics")
		}
		return nil
	}

	if head.Type != reader.NodeSymbol {
		return nil
	}
	headVal := head.Value

	if isPutCall(node) {
		if len(node.Children) >= 3 {
			valueArg := node.Children[2]
			if valueArg.Type != reader.NodeMap {
				sentinel := isDirectSentinel(valueArg)
				if sentinel != "" {
					return rules.SetContextualFindingWithEvidence(&rules.Finding{
						RuleID:   r.ID,
						Message:  fmt.Sprintf("Sentinel value %s in %s: prefer (close! ch) so that (<! ch) returns nil; avoid custom sentinels.", sentinel, headVal),
						Filepath: filepath,
						Location: node.Location,
						Severity: rules.SeverityHint,
						Tags:     append(rules.ContextualTags(context), "low-confidence", "producer-only"),
					}, "A sentinel may be a legitimate part of the channel's domain protocol.", "channel-domain-protocol", "close-semantics")
				}
			}
		}
		return nil
	}

	comparisonReadsChannel := false
	for _, child := range node.Children[1:] {
		if isChannelTakeForm(child) {
			comparisonReadsChannel = true
			break
		}
	}
	if rules.CallResolvesTo(node, "clojure.core/not=", "clojure.core/=") && validComparisonArity(node) && (isInsideGoBlock(context) || comparisonReadsChannel) {
		var sentinel string
		for _, child := range node.Children[1:] {
			if s := isDirectSentinel(child); s != "" {
				sentinel = s
				break
			}
			if child.Type == reader.NodeNumber && child.Value == "-1" {
				sentinel = "-1"
				break
			}
		}
		if sentinel != "" {
			return rules.SetContextualFindingWithEvidence(&rules.Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Comparison with sentinel %s: prefer (close! ch) so that (<! ch) returns nil; use (when-let [e (<! ch)] ...) when closed.", sentinel),
				Filepath: filepath,
				Location: node.Location,
				Severity: rules.ContextualSeverity(context, r.Severity),
				Tags:     rules.ContextualTags(context),
			}, "A sentinel may be a legitimate part of the channel's domain protocol.", "channel-domain-protocol", "close-semantics")
		}
	}

	if rules.CallResolvesTo(node, "clojure.core/contains?", "clojure.core/get") && validMapQueryArity(node) && isInsideGoBlock(context) {
		if len(node.Children) >= 3 {
			keyArg := node.Children[2]
			if sentinel := isDirectSentinel(keyArg); sentinel != "" {
				return rules.SetContextualFindingWithEvidence(&rules.Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Checking sentinel key %s: prefer closing channels and checking for nil.", sentinel),
					Filepath: filepath,
					Location: node.Location,
					Severity: rules.ContextualSeverity(context, r.Severity),
					Tags:     rules.ContextualTags(context),
				}, "A sentinel may be a legitimate part of the channel's domain protocol.", "channel-domain-protocol", "close-semantics")
			}
		}
	}

	return nil
}

func isInsideProtocolDeclaration(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if ancestor == nil || ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		switch strings.TrimPrefix(ancestor.Children[0].Value, "clojure.core/") {
		case "defprotocol", "extend-protocol", "extend-type", "definterface", "proxy", "reify":
			return true
		}
	}
	return false
}

func isInsideGoBlock(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, ancestor := range ancestors {
		if rules.CallResolvesTo(ancestor, "clojure.core.async/go", "clojure.core.async/go-loop") {
			return true
		}
	}
	return false
}

func isDirectSentinel(node *reader.RichNode) string {
	if node == nil {
		return ""
	}
	if node.Type == reader.NodeKeyword || node.Type == reader.NodeString {
		if isSentinelKeyword(node.Value) {
			return node.Value
		}
	}
	return ""
}

func isPutCall(node *reader.RichNode) bool {
	if !rules.CallResolvesTo(node, "clojure.core.async/put!", "clojure.core.async/>!", "clojure.core.async/>!!") {
		return false
	}
	return validPutArity(node)
}

func isTakeCall(node *reader.RichNode) bool {
	resolved := rules.ResolvedCall(node)
	if resolved == nil || !rules.CallResolvesTo(node, "clojure.core.async/<!", "clojure.core.async/<!!") {
		return false
	}
	known, valid := semantics.CanonicalArityValid(resolved.CanonicalName, len(node.Children)-1)
	return known && valid
}

func validPutArity(node *reader.RichNode) bool {
	if node == nil || len(node.Children) < 3 {
		return false
	}
	resolved := rules.ResolvedCall(node)
	if resolved == nil {
		return false
	}
	known, valid := semantics.CanonicalArityValid(resolved.CanonicalName, len(node.Children)-1)
	return known && valid
}

func validComparisonArity(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	resolved := rules.ResolvedCall(node)
	if resolved == nil {
		return false
	}
	known, valid := semantics.CanonicalArityValid(resolved.CanonicalName, len(node.Children)-1)
	return known && valid
}

func validMapQueryArity(node *reader.RichNode) bool {
	if node == nil || len(node.Children) < 3 {
		return false
	}
	resolved := rules.ResolvedCall(node)
	if resolved == nil {
		return false
	}
	known, valid := semantics.CanonicalArityValid(resolved.CanonicalName, len(node.Children)-1)
	return known && valid
}

func isChannelTakeForm(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return false
	}
	return isTakeCall(node)
}

var sentinelStems = []string{
	"done", "end", "eof", "close", "stop", "exit",
	"complete", "finish", "eos", "poison", "bye", "quit", "terminat", "terminate",
	"closed", "finished", "completed",
	"shutdown",
}

func isSentinelKeyword(v string) bool {
	local := keywordLocalName(v)
	if local == "" {
		return false
	}
	lower := strings.ToLower(local)
	for _, stem := range sentinelStems {
		if lower == stem || strings.HasPrefix(lower, stem+"-") || strings.HasPrefix(lower, stem+"_") || strings.HasPrefix(lower, stem+"/") {
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
