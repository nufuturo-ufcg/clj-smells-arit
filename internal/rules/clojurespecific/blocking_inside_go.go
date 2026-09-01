package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

type BlockingInsideGoRule struct{ rules.Rule }

func (r *BlockingInsideGoRule) Meta() rules.Rule { return r.Rule }

var knownBlockingCalls = map[string]struct{}{
	"clojure.core.async/<!!": {}, "clojure.core.async/>!!": {},
	"clojure.core.async/alts!!": {}, "clojure.core.async/alt!!": {},
	"Thread/sleep": {}, "java.lang.Thread/sleep": {},
	"clojure.core/slurp": {}, "clojure.core/spit": {}, "clojure.core/await": {},
	"clojure.java.io/reader": {}, "clojure.java.io/writer": {},
	"clojure.java.io/input-stream": {}, "clojure.java.io/output-stream": {},
	"clojure.java.jdbc/execute!": {}, "clojure.java.jdbc/query": {}, "clojure.java.jdbc/insert!": {},
	"next.jdbc/execute!": {}, "next.jdbc.sql/query": {}, "next.jdbc.sql/insert!": {},
	"clj-http.client/get": {}, "clj-http.client/post": {}, "clj-http.client/put": {}, "clj-http.client/request": {},
	".readLine": {}, ".acquire": {}, "java.net.Socket.": {},
}

func resolvedCanonical(node *reader.RichNode) string {
	resolved := rules.ResolvedCall(node)
	if resolved == nil || resolved.Kind == reader.ResolutionUnresolved || resolved.Kind == reader.ResolutionLocal {
		return ""
	}
	return resolved.CanonicalName
}

func isBlockingCall(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	head := node.Children[0]
	if head.Resolution != nil && head.Resolution.Kind == reader.ResolutionLocal {
		return false
	}

	canonical := resolvedCanonical(node)
	if canonical == "" {
		return false
	}
	_, ok := knownBlockingCalls[canonical]
	return ok && blockingCallHasValidArity(canonical, len(node.Children)-1)
}

func blockingCallHasValidArity(canonical string, argumentCount int) bool {
	if known, valid := semantics.CanonicalArityValid(canonical, argumentCount); known {
		return valid
	}
	return argumentCount >= 1
}

func isGoBlock(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	canonical := resolvedCanonical(node)
	return canonical == "clojure.core.async/go" || canonical == "clojure.core.async/go-loop"
}

func isDeferredBoundary(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	switch node.Type {
	case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard, reader.NodeFnLiteral:
		return true
	}
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	canonical := resolvedCanonical(node)
	switch canonical {
	case "clojure.core.async/thread", "clojure.core/future", "clojure.core/delay", "clojure.core/lazy-seq":
		return true
	}
	head := node.Children[0]
	if head.Type == reader.NodeSymbol {
		val := head.Value
		if slash := strings.LastIndex(val, "/"); slash >= 0 {
			val = val[slash+1:]
		}
		switch val {
		case "fn", "fn*", "letfn", "thread", "future", "delay", "lazy-seq", "bound-fn", "bound-fn*":
			return true
		}
	}
	return false
}

func resolvedFunctionBody(call *reader.RichNode) []*reader.RichNode {
	if call == nil || call.Type != reader.NodeList || len(call.Children) == 0 {
		return nil
	}
	head := call.Children[0]
	if head == nil || head.Resolution == nil || head.Resolution.Kind != reader.ResolutionLocal ||
		head.ResolvedDefinition == nil {
		return nil
	}
	definition := head.ResolvedDefinition
	if definition.Type != reader.NodeList || len(definition.Children) < 3 ||
		definition.Children[0].Type != reader.NodeSymbol {
		return nil
	}
	switch definition.Children[0].Value {
	case "defn", "defn-":
	default:
		return nil
	}

	idx := 2
	if definition.Children[idx].Type == reader.NodeString {
		idx++
	}
	if idx < len(definition.Children) && definition.Children[idx].Type == reader.NodeMap {
		idx++
	}
	if idx >= len(definition.Children) {
		return nil
	}
	if definition.Children[idx].Type == reader.NodeVector {
		return definition.Children[idx+1:]
	}

	callArity := len(call.Children) - 1
	for _, arity := range definition.Children[idx:] {
		if arity == nil || arity.Type != reader.NodeList || len(arity.Children) < 2 ||
			arity.Children[0].Type != reader.NodeVector {
			continue
		}
		if parameterVectorAcceptsArity(arity.Children[0], callArity) {
			return arity.Children[1:]
		}
	}
	return nil
}

func parameterVectorAcceptsArity(params *reader.RichNode, callArity int) bool {
	if params == nil || params.Type != reader.NodeVector {
		return false
	}
	fixed := 0
	variadic := false
	for _, param := range params.Children {
		if param.Type == reader.NodeSymbol && param.Value == "&" {
			variadic = true
			break
		}
		fixed++
	}
	if variadic {
		return callArity >= fixed
	}
	return callArity == fixed
}

const maxBlockingCallDepth = 6

func findBlockingCall(node *reader.RichNode, definitions map[*reader.RichNode]bool, depth int) *reader.RichNode {
	if node == nil || depth > maxBlockingCallDepth || isDeferredBoundary(node) {
		return nil
	}
	if node.Type == reader.NodeList {
		if isBlockingCall(node) {
			return node
		}
		if body := resolvedFunctionBody(node); len(body) > 0 {
			definition := node.Children[0].ResolvedDefinition
			if !definitions[definition] {
				definitions[definition] = true
				for _, bodyNode := range body {
					if found := findBlockingCall(bodyNode, definitions, depth+1); found != nil {
						return found
					}
				}
				delete(definitions, definition)
			}
		}
	}
	for _, child := range node.Children {
		if found := findBlockingCall(child, definitions, depth); found != nil {
			return found
		}
	}
	return nil
}

func (r *BlockingInsideGoRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if !isGoBlock(node) {
		return nil
	}
	if ancestors, ok := context["ancestorNodes"].([]*reader.RichNode); ok {
		for _, ancestor := range ancestors {
			if channel, _ := coreAsyncSingleValueChannel(ancestor); channel != nil {
				return nil
			}
		}
	}
	var blocking *reader.RichNode
	for _, child := range node.Children[1:] {
		if blocking = findBlockingCall(child, make(map[*reader.RichNode]bool), 0); blocking != nil {
			break
		}
	}
	if blocking == nil {
		return nil
	}
	return &rules.Finding{
		RuleID:   r.ID,
		Message:  fmt.Sprintf("Blocking function detected within the GO block %s.", node.Children[0].Value),
		Filepath: filepath, Location: blocking.Location,
		Severity: rules.ContextualSeverity(context, r.Severity),
		Tags:     rules.ContextualTags(context),
	}
}

func init() {
	rules.RegisterRule(&BlockingInsideGoRule{Rule: rules.Rule{
		ID: "blocking-inside-go", Name: "Blocking Inside GO",
		Description: "Using blocking functions directly or through a resolved local function within a GO block violates its non-blocking purpose.",
		Severity:    rules.SeverityWarning,
	}})
}
