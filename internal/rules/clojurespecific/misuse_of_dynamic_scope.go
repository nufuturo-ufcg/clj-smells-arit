package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type MisuseOfDynamicScopeRule struct {
	rules.Rule
}

func (r *MisuseOfDynamicScopeRule) Meta() rules.Rule {
	return r.Rule
}

func isAllowedDynamicVar(name string) bool {
	switch name {
	case "*out*", "*err*", "*in*", "*ns*", "*warn-on-reflection*", "*file*", "*compile-path*", "*command-line-args*", "*agent*", "*math-context*", "*print-length*", "*print-level*", "*data-readers*", "*default-data-reader-fn*", "*read-eval*":
		return true
	}
	return false
}

func isExplicitResourceVar(name string) bool {
	lower := strings.ToLower(name)
	lower = strings.TrimPrefix(lower, "*")
	lower = strings.TrimSuffix(lower, "*")

	parts := strings.Split(lower, "-")
	keywords := map[string]bool{
		"db": true, "conn": true, "pool": true, "tx": true, "transaction": true,
		"client": true, "producer": true, "socket": true, "ds": true,
		"redis": true, "kafka": true, "s3": true, "http": true,
		"writer": true, "session": true,
	}

	for _, part := range parts {
		if keywords[part] {
			return true
		}
	}
	return false
}

func (r *MisuseOfDynamicScopeRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	if node == nil {
		return nil
	}
	allowedVars := rules.RuleSettingStringSlice(context, r.Meta().ID, "allowed_vars")
	if node.Type == reader.NodeSymbol {
		name := node.Value
		lazyBoundary := hasLazyDynamicBoundary(context) && lazyBoundaryDefersRead(context)
		if !isKnownDynamicName(name, context) || rules.MatchesConfiguredName(name, allowedVars) ||
			dynamicReadInSyntaxQuoteTemplate(context) ||
			// A lexical binding shadows the var; a file-level def is the var.
			(node.Resolution != nil && node.Resolution.Kind == reader.ResolutionLocal && node.Resolution.Lexical) ||
			(!hasAsyncBoundary(context) && !lazyBoundary && !readInsideBindingWithinGo(context)) ||
			(!lazyBoundary && argumentEvaluatedEagerly(node, context)) || dynamicDefinitionNode(node, context) ||
			dynamicBindingNameNode(node, context) || dynamicLexicalBindingNameNode(node, context) {
			return nil
		}
		return rules.SetContextualFindingWithEvidence(&rules.Finding{
			RuleID:   r.Meta().ID,
			Message:  fmt.Sprintf("Custom dynamic variable `%s` is read across an asynchronous or lazy boundary. Prefer explicit data flow or preserve the binding around the complete operation.", name),
			Filepath: filepath,
			Location: node.Location,
			Severity: rules.ContextualSeverity(context, r.Meta().Severity),
		}, "The read may depend on binding propagation, lazy realization, or worker execution semantics.", "dynamic-binding-propagation", "execution-boundary-contract")
	}
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	firstElement := node.Children[0]
	if firstElement.Type != reader.NodeSymbol {
		return nil
	}
	if firstElement.Value == "def" {
		if definition, name := dynamicDefinitionSymbol(node); name != "" && isDeclaredDynamicVar(node, definition) {
			registerDynamicVar(context, name)
		}
	}
	if !hasWorkerBoundary(context) {
		return nil
	}

	if firstElement.Value == "def" && len(node.Children) > 1 {
		definition, name := dynamicDefinitionSymbol(node)
		if definition == nil {
			return nil
		}
		loc := definition.Location
		if name != "" && isKnownDynamicName(name, context) {
			if !isAllowedDynamicVar(name) && !isExplicitResourceVar(name) && !rules.MatchesConfiguredName(name, allowedVars) {
				return rules.SetContextualFindingWithEvidence(&rules.Finding{
					RuleID:   r.Meta().ID,
					Message:  fmt.Sprintf("Defining custom dynamic variable `%s`. Dynamic scope is often misused for passing business context, which obfuscates data flow and causes bugs in async boundaries.", name),
					Filepath: filepath,
					Location: loc,
					Severity: rules.ContextualSeverity(context, r.Meta().Severity),
				}, "A custom dynamic variable may be a deliberate contextual API; the AST does not prove that business data is being passed incorrectly.", "dynamic-var-contract", "business-context")
			}
		}
	}

	// A binding created inside a worker is local configuration for that worker;
	// it does not by itself prove that a parent binding was lost. The previous
	// implementation reported every binding below future/send, which produced
	// false positives for explicit worker-local configuration. Keep the check
	// only for core.async state-machine boundaries, where parking can resume
	// execution outside the lexical binding scope.
	if firstElement.Value == "binding" && len(node.Children) > 1 {
		if !hasProblematicAsyncBoundary(context) {
			return nil
		}
		bindings := node.Children[1]
		if bindings.Type == reader.NodeVector {
			for i := 0; i < len(bindings.Children); i += 2 {
				varNode := bindings.Children[i]
				if varNode.Type == reader.NodeSymbol {
					name := varNode.Value
					if !isAllowedDynamicVar(name) && !isExplicitResourceVar(name) && !rules.MatchesConfiguredName(name, allowedVars) {
						severity := rules.ContextualSeverity(context, r.Meta().Severity)
						tags := rules.ContextualTags(context)
						if i+1 < len(bindings.Children) && bindings.Children[i+1].Type == reader.NodeNil {
							severity = rules.SeverityHint
							tags = append(tags, "explicit-reset")
						}
						finding := &rules.Finding{
							RuleID:   r.Meta().ID,
							Message:  fmt.Sprintf("Binding custom dynamic variable `%s` across a core.async boundary. Prefer explicit data flow or preserve the binding around the complete operation.", name),
							Filepath: filepath,
							Location: varNode.Location,
							Severity: severity,
							Tags:     tags,
						}
						return rules.SetContextualFindingWithEvidence(finding, "Propagating a custom binding through core.async depends on execution semantics and how state is resumed.", "binding-propagation", "async-resume-semantics")
					}
				}
			}
		}
	}

	return nil
}

func dynamicDefinitionSymbol(node *reader.RichNode) (*reader.RichNode, string) {
	if node == nil {
		return nil, ""
	}
	for i := 1; i < len(node.Children); i++ {
		if node.Children[i].Type == reader.NodeSymbol {
			return node.Children[i], node.Children[i].Value
		}
	}
	return nil, ""
}

func isCustomDynamicName(name string) bool {
	return len(name) > 2 && strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") &&
		!isAllowedDynamicVar(name) && !isExplicitResourceVar(name)
}

func isKnownDynamicName(name string, context map[string]interface{}) bool {
	if isCustomDynamicName(name) {
		return true
	}
	dynamicVars, _ := context["dynamic-vars"].(map[string]bool)
	return dynamicVars[name]
}

func registerDynamicVar(context map[string]interface{}, name string) {
	dynamicVars, _ := context["dynamic-vars"].(map[string]bool)
	if dynamicVars == nil {
		dynamicVars = map[string]bool{}
		context["dynamic-vars"] = dynamicVars
	}
	dynamicVars[name] = true
}

func isDeclaredDynamicVar(node, definition *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if metadataContainsMarker(node.Metadata, ":dynamic") || metadataContainsMarker(node.Metadata, "dynamic") {
		return true
	}
	if definition == nil {
		return false
	}
	if metadataContainsMarker(definition.Metadata, ":dynamic") || metadataContainsMarker(definition.Metadata, "dynamic") {
		return true
	}
	for _, child := range node.Children {
		if child == definition {
			break
		}
		if metadataContainsMarker(child, ":dynamic") || metadataContainsMarker(child, "dynamic") {
			return true
		}
	}
	return false
}

func metadataContainsMarker(node *reader.RichNode, marker string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeKeyword && node.Value == marker {
		return true
	}
	if metadataContainsMarker(node.Metadata, marker) {
		return true
	}
	for _, child := range node.Children {
		if metadataContainsMarker(child, marker) {
			return true
		}
	}
	return false
}

func dynamicDefinitionNode(node *reader.RichNode, context map[string]interface{}) bool {
	parent, _ := context["parent"].(*reader.RichNode)
	return parent != nil && parent.Type == reader.NodeList && len(parent.Children) > 1 &&
		parent.Children[0] != nil && parent.Children[0].Value == "def" && parent.Children[1] == node
}

func dynamicBindingNameNode(node *reader.RichNode, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) < 2 {
		return false
	}
	bindingVector := ancestors[len(ancestors)-1]
	bindingForm := ancestors[len(ancestors)-2]
	if bindingVector == nil || bindingVector.Type != reader.NodeVector || bindingForm == nil || bindingForm.Type != reader.NodeList || len(bindingForm.Children) == 0 || bindingForm.Children[0].Value != "binding" {
		return false
	}
	for index, child := range bindingVector.Children {
		if child == node {
			return index%2 == 0
		}
	}
	return false
}

func dynamicLexicalBindingNameNode(node *reader.RichNode, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) < 2 {
		return false
	}
	bindingVector := ancestors[len(ancestors)-1]
	bindingForm := ancestors[len(ancestors)-2]
	if bindingVector == nil || bindingVector.Type != reader.NodeVector || bindingForm == nil ||
		bindingForm.Type != reader.NodeList || len(bindingForm.Children) == 0 || bindingForm.Children[0].Type != reader.NodeSymbol {
		return false
	}
	switch bindingForm.Children[0].Value {
	case "let", "let*", "loop", "loop*", "if-let", "when-let", "with-open", "binding", "defn", "defn-", "fn", "fn*", "letfn":
	default:
		return false
	}
	for index, child := range bindingVector.Children {
		if child == node {
			return index%2 == 0
		}
	}
	return false
}

func hasLazyDynamicBoundary(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	hasBinding := false
	realizedInside := false
	for _, form := range enclosing {
		form = strings.TrimPrefix(form, "clojure.core/")
		if form == "binding" {
			hasBinding = true
			realizedInside = false
			continue
		}
		if hasBinding && isEagerConsumerForm(form) {
			// A sequência preguiçosa é realizada dentro do binding.
			realizedInside = true
			continue
		}
		switch form {
		case "map", "mapcat", "filter", "remove", "keep", "map-indexed", "filterv", "for", "lazy-seq", "sequence":
			if hasBinding && !realizedInside {
				return true
			}
		}
	}
	return false
}

func isEagerConsumerForm(form string) bool {
	switch form {
	case "into", "vec", "set", "doall", "dorun", "mapv", "filterv", "reduce", "transduce", "run!",
		"apply", "count", "frequencies", "group-by", "sort", "sort-by", "doseq", "str/join",
		"clojure.string/join", "zipmap", "every?", "some", "reduce-kv", "into-array", "to-array":
		return true
	}
	return false
}

func hasAsyncBoundary(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	for _, form := range enclosing {
		if isAsyncBoundaryForm(form) {
			return true
		}
	}
	return false
}

// isAsyncBoundaryForm recognizes forms that run code on another thread
// WITHOUT conveying dynamic bindings. future, future-call, pmap, send,
// send-off, agent actions, core.async go/go-loop and thread all convey the
// caller's bindings (binding-conveyor-fn), so a read inside them sees the
// binding that was active at creation; reporting them produced false positives.
// Raw executors and threads do not convey bindings.
func isAsyncBoundaryForm(form string) bool {
	switch form {
	case ".submit", ".execute", ".start", "ExecutorService/submit", "ExecutorService/execute",
		"Thread.", "java.lang.Thread.", "new Thread":
		return true
	}
	return false
}

// argumentEvaluatedEagerly indica que o símbolo é argumento direto de uma forma
// cujos argumentos são avaliados na thread atual (send, send-off, pmap, agent,
// future-call, .submit, .execute); só a função passada roda do outro lado.
func argumentEvaluatedEagerly(node *reader.RichNode, context map[string]interface{}) bool {
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil || parent.Type != reader.NodeList || len(parent.Children) < 2 ||
		parent.Children[0] == nil || parent.Children[0].Type != reader.NodeSymbol || parent.Children[0] == node {
		return false
	}
	head := strings.TrimPrefix(parent.Children[0].Value, "clojure.core/")
	switch {
	case head == "send", head == "send-off", head == "agent", head == ".submit", head == ".execute",
		head == "pmap" || strings.HasSuffix(head, "/pmap"),
		head == "future-call" || strings.HasSuffix(head, "/future-call"):
		return true
	}
	return false
}

// hasWorkerBoundary keeps the broad list of worker forms for the definition
// and binding checks (a def or a binding created inside a worker), which are
// independent of binding conveyance.
func hasWorkerBoundary(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	for _, form := range enclosing {
		if isAsyncBoundaryForm(form) {
			return true
		}
		form = strings.TrimPrefix(form, "clojure.core/")
		switch {
		case form == "go" || strings.HasSuffix(form, "/go"),
			form == "go-loop" || strings.HasSuffix(form, "/go-loop"),
			form == "thread" || strings.HasSuffix(form, "/thread"),
			form == "future" || strings.HasSuffix(form, "/future"),
			form == "future-call" || strings.HasSuffix(form, "/future-call"),
			form == "pmap" || strings.HasSuffix(form, "/pmap"),
			form == "send", form == "send-off", form == "agent":
			return true
		}
	}
	return false
}

// readInsideBindingWithinGo: a binding established inside a go block can be
// lost at a park point, so reads under it are still reported.
func readInsideBindingWithinGo(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	inGo := false
	for _, form := range enclosing {
		form = strings.TrimPrefix(form, "clojure.core/")
		if form == "go" || form == "go-loop" || strings.HasSuffix(form, "/go") || strings.HasSuffix(form, "/go-loop") {
			inGo = true
			continue
		}
		if inGo && form == "binding" {
			return true
		}
	}
	return false
}

func hasProblematicAsyncBoundary(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	for _, form := range enclosing {
		form = strings.TrimPrefix(form, "clojure.core/")
		switch {
		case form == "go" || form == "go-loop" || strings.HasSuffix(form, "/go") || strings.HasSuffix(form, "/go-loop"):
			return true
		}
	}
	return false
}

func init() {
	defaultRule := &MisuseOfDynamicScopeRule{
		Rule: rules.Rule{
			ID:          "misuse-of-dynamic-scope",
			Name:        "Misuse of Dynamic Scope",
			Description: "Flags the definition and usage of custom dynamic variables (*var*), which are often misused to pass business parameters implicitly, hiding dependencies and breaking thread boundaries.",
			Severity:    rules.SeverityWarning,
		},
	}
	rules.RegisterRule(defaultRule)
}

// dynamicReadInSyntaxQuoteTemplate reports a read inside a syntax-quote
// template (not under an unquote): it is generated code, evaluated wherever
// the expansion lands, not at this point.
func dynamicReadInSyntaxQuoteTemplate(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	for index := len(ancestors) - 1; index >= 0; index-- {
		if ancestor := ancestors[index]; ancestor != nil {
			switch ancestor.Type {
			case reader.NodeUnquote, reader.NodeUnquoteSplice:
				return false
			case reader.NodeSyntaxQuote:
				return true
			}
		}
	}
	return false
}

// lazyBoundaryDefersRead requires that some lazy sequence operation really
// defers the read: map/mapcat/filter... evaluate their arguments eagerly and
// only delay the body of the function they receive, so a read in an argument
// such as (mapcat (if *v* f identity) xs) happens right away. A lazy step
// realized by a later eager step of the same threading form (->> ... vec) is
// realized where it is created.
func lazyBoundaryDefersRead(context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	insideFn := false
	for index := len(ancestors) - 1; index >= 0; index-- {
		ancestor := ancestors[index]
		if ancestor == nil {
			continue
		}
		if ancestor.Type == reader.NodeFnLiteral {
			insideFn = true
			continue
		}
		if ancestor.Type != reader.NodeList || len(ancestor.Children) == 0 || ancestor.Children[0] == nil ||
			ancestor.Children[0].Type != reader.NodeSymbol {
			continue
		}
		head := strings.TrimPrefix(ancestor.Children[0].Value, "clojure.core/")
		switch head {
		case "fn", "fn*", "letfn":
			insideFn = true
		case "for", "lazy-seq", "lazy-cat":
			if !lazyStepRealizedByThreading(ancestors, index) {
				return true
			}
		case "map", "mapcat", "filter", "remove", "keep", "map-indexed", "keep-indexed", "sequence":
			if insideFn && !lazyStepRealizedByThreading(ancestors, index) {
				return true
			}
		}
	}
	return false
}

func lazyStepRealizedByThreading(ancestors []*reader.RichNode, index int) bool {
	if index == 0 {
		return false
	}
	step, thread := ancestors[index], ancestors[index-1]
	if thread == nil || thread.Type != reader.NodeList || len(thread.Children) < 3 || thread.Children[0] == nil {
		return false
	}
	switch strings.TrimPrefix(thread.Children[0].Value, "clojure.core/") {
	case "->", "->>":
	default:
		return false
	}
	seen := false
	for _, child := range thread.Children[2:] {
		if child == step {
			seen = true
			continue
		}
		if !seen {
			continue
		}
		name := ""
		if child != nil && child.Type == reader.NodeSymbol {
			name = child.Value
		} else if child != nil && child.Type == reader.NodeList && len(child.Children) > 0 && child.Children[0] != nil {
			name = child.Children[0].Value
		}
		if isEagerConsumerForm(strings.TrimPrefix(name, "clojure.core/")) {
			return true
		}
	}
	return false
}
