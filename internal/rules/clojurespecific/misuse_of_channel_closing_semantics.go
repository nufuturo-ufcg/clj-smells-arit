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
				// A sentinel is only a misuse on a data channel: a channel that
				// also carries ordinary values in the same top-level form. A
				// single signal on a control channel (shutdown, done, lock token)
				// is a legitimate protocol.
				if sentinel != "" && channelAlsoCarriesData(node, context) {
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
	// A comparison is only evidence of a channel sentinel when the compared
	// value comes from a channel take, directly or through a symbol bound to a
	// take. Being inside a go block is not enough: (= -1 (:season payload))
	// compares domain data.
	if rules.CallResolvesTo(node, "clojure.core/not=", "clojure.core/=") && validComparisonArity(node) &&
		(comparisonReadsChannel || comparesTakeBoundSymbol(node, context)) {
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

func comparesTakeBoundSymbol(node *reader.RichNode, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for _, child := range node.Children[1:] {
		if child == nil {
			continue
		}
		if child.Type == reader.NodeSymbol && symbolBoundToTake(child.Value, ancestors) {
			return true
		}
		// A field of the taken value: (:status msg) or (get msg :status).
		if child.Type == reader.NodeList && len(child.Children) >= 2 && child.Children[0] != nil {
			target := child.Children[1]
			if child.Children[0].Type == reader.NodeKeyword ||
				(child.Children[0].Type == reader.NodeSymbol && rules.CallResolvesTo(child, "clojure.core/get")) {
				if target != nil && target.Type == reader.NodeSymbol && symbolBoundToTake(target.Value, ancestors) {
					return true
				}
			}
		}
	}
	return false
}

// symbolBoundToTake walks the enclosing binding forms from the innermost one
// and reports whether the nearest binding of symbol is a channel take.
func symbolBoundToTake(symbol string, ancestors []*reader.RichNode) bool {
	for i := len(ancestors) - 1; i >= 0; i-- {
		form := ancestors[i]
		if form == nil || form.Type != reader.NodeList || len(form.Children) < 2 ||
			form.Children[0] == nil || form.Children[0].Type != reader.NodeSymbol {
			continue
		}
		head := form.Children[0].Value
		if j := strings.LastIndex(head, "/"); j >= 0 {
			head = head[j+1:]
		}
		switch head {
		case "let", "let*", "loop", "loop*", "when-let", "if-let", "when-some", "if-some", "go-loop":
		default:
			continue
		}
		bindings := form.Children[1]
		if bindings == nil || bindings.Type != reader.NodeVector {
			continue
		}
		for k := len(bindings.Children) - 2; k >= 0; k -= 2 {
			name := bindings.Children[k]
			if name != nil && name.Type == reader.NodeSymbol && name.Value == symbol {
				return isChannelTakeForm(bindings.Children[k+1])
			}
		}
	}
	return false
}

func channelAlsoCarriesData(put *reader.RichNode, context map[string]interface{}) bool {
	channel := put.Children[1]
	if channel == nil || channel.Type != reader.NodeSymbol {
		return false
	}
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) == 0 {
		return false
	}
	// A channel defined at top level is shared by the whole file; a local
	// channel is confined to its top-level form.
	scope := []*reader.RichNode{ancestors[0]}
	if channel.ResolvedDefinition != nil && channel.ResolvedDefinition.Type == reader.NodeList &&
		len(channel.ResolvedDefinition.Children) > 0 && channel.ResolvedDefinition.Children[0] != nil &&
		(channel.ResolvedDefinition.Children[0].Value == "def" || channel.ResolvedDefinition.Children[0].Value == "defonce") {
		if roots, ok := context["file-roots"].([]*reader.RichNode); ok {
			scope = roots
		}
	}
	var found bool
	// guarded holds symbols known, in the current branch, to be one of a set
	// of literal keywords: (cond (#{:stopped :timeout} block) (put! ch block)).
	var walk func(node *reader.RichNode, guarded map[string]bool)
	walk = func(node *reader.RichNode, guarded map[string]bool) {
		if node == nil || found {
			return
		}
		if node != put && isPutCall(node) && len(node.Children) >= 3 {
			target, value := node.Children[1], node.Children[2]
			if target != nil && target.Type == reader.NodeSymbol && target.Value == channel.Value &&
				isDirectSentinel(value) == "" && !(value != nil && value.Type == reader.NodeKeyword) &&
				!(value != nil && value.Type == reader.NodeSymbol && guarded[value.Value]) {
				found = true
				return
			}
		}
		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
			switch strings.TrimPrefix(node.Children[0].Value, "clojure.core/") {
			case "cond":
				for i := 1; i < len(node.Children); i++ {
					if i%2 == 0 {
						if sym := keywordGuardSymbol(node.Children[i-1]); sym != "" {
							walk(node.Children[i], withGuard(guarded, sym))
							continue
						}
					}
					walk(node.Children[i], guarded)
				}
				return
			case "if", "when":
				if len(node.Children) >= 3 {
					if sym := keywordGuardSymbol(node.Children[1]); sym != "" {
						walk(node.Children[1], guarded)
						walk(node.Children[2], withGuard(guarded, sym))
						for _, rest := range node.Children[3:] {
							walk(rest, guarded)
						}
						return
					}
				}
			}
		}
		for _, child := range node.Children {
			walk(child, guarded)
		}
	}
	for _, root := range scope {
		walk(root, map[string]bool{})
	}
	return found
}

func withGuard(guarded map[string]bool, symbol string) map[string]bool {
	next := make(map[string]bool, len(guarded)+1)
	for k, v := range guarded {
		next[k] = v
	}
	next[symbol] = true
	return next
}

// keywordGuardSymbol returns x for tests (#{:a :b} x), (= x :a) or (= :a x).
func keywordGuardSymbol(test *reader.RichNode) string {
	if test == nil || test.Type != reader.NodeList || len(test.Children) < 2 {
		return ""
	}
	head := test.Children[0]
	if head != nil && head.Type == reader.NodeSet && len(test.Children) == 2 && len(head.Children) > 0 {
		for _, element := range head.Children {
			if element == nil || element.Type != reader.NodeKeyword {
				return ""
			}
		}
		if arg := test.Children[1]; arg != nil && arg.Type == reader.NodeSymbol {
			return arg.Value
		}
		return ""
	}
	if head != nil && head.Type == reader.NodeSymbol && strings.TrimPrefix(head.Value, "clojure.core/") == "=" && len(test.Children) == 3 {
		a, b := test.Children[1], test.Children[2]
		if a != nil && b != nil && a.Type == reader.NodeSymbol && b.Type == reader.NodeKeyword {
			return a.Value
		}
		if a != nil && b != nil && b.Type == reader.NodeSymbol && a.Type == reader.NodeKeyword {
			return b.Value
		}
	}
	return ""
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
