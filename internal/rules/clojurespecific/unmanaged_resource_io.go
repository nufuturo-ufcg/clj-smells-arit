package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type UnmanagedResourceIORule struct {
	rules.Rule
}

func (r *UnmanagedResourceIORule) Meta() rules.Rule {
	return r.Rule
}

// These are operations for which the returned value itself owns a resource
// that normally must be closed. Deliberately do not infer this from names:
// symbols such as context->app-connection and byte-stream-from-buffers do not
// prove that a new Closeable was created at the call site.
var unmanagedResourceCreators = map[string]struct{}{
	"clojure.java.io/reader":        {},
	"clojure.java.io/writer":        {},
	"clojure.java.io/input-stream":  {},
	"clojure.java.io/output-stream": {},

	"java.io.FileReader.":       {},
	"FileReader.":               {},
	"java.io.FileWriter.":       {},
	"FileWriter.":               {},
	"java.io.FileInputStream.":  {},
	"FileInputStream.":          {},
	"java.io.FileOutputStream.": {},
	"FileOutputStream.":         {},
	"java.io.RandomAccessFile.": {},
	"RandomAccessFile.":         {},

	"java.net.Socket.":         {},
	"Socket.":                  {},
	"java.net.ServerSocket.":   {},
	"ServerSocket.":            {},
	"java.net.DatagramSocket.": {},
	"DatagramSocket.":          {},

	"java.io.BufferedReader.":       {},
	"BufferedReader.":               {},
	"java.io.BufferedWriter.":       {},
	"BufferedWriter.":               {},
	"java.io.InputStreamReader.":    {},
	"InputStreamReader.":            {},
	"java.io.OutputStreamWriter.":   {},
	"OutputStreamWriter.":           {},
	"java.io.BufferedInputStream.":  {},
	"BufferedInputStream.":          {},
	"java.io.BufferedOutputStream.": {},
	"BufferedOutputStream.":         {},

	"java.util.zip.ZipFile.":          {},
	"ZipFile.":                        {},
	"java.util.jar.JarFile.":          {},
	"JarFile.":                        {},
	"java.util.zip.GZIPInputStream.":  {},
	"GZIPInputStream.":                {},
	"java.util.zip.GZIPOutputStream.": {},
	"GZIPOutputStream.":               {},
	"java.util.zip.ZipInputStream.":   {},
	"ZipInputStream.":                 {},
	"java.util.zip.ZipOutputStream.":  {},
	"ZipOutputStream.":                {},

	"java.nio.file.Files/newInputStream":  {},
	"java.nio.file.Files/newOutputStream": {},
	"java.nio.file.Files/lines":           {},
	"Files/newInputStream":                {},
	"Files/newOutputStream":               {},
	"Files/lines":                         {},

	"java.sql.DriverManager/getConnection": {},
	"DriverManager/getConnection":          {},
	".getConnection":                       {},
	"getConnection":                        {},
	".getInputStream":                      {},
	"getInputStream":                       {},
}

var unmanagedInMemoryCreators = map[string]struct{}{
	"java.io.StringReader.":          {},
	"StringReader.":                  {},
	"java.io.StringWriter.":          {},
	"StringWriter.":                  {},
	"java.io.ByteArrayInputStream.":  {},
	"ByteArrayInputStream.":          {},
	"java.io.ByteArrayOutputStream.": {},
	"ByteArrayOutputStream.":         {},
	"java.io.CharArrayReader.":       {},
	"CharArrayReader.":               {},
	"java.io.CharArrayWriter.":       {},
	"CharArrayWriter.":               {},
}

// Calls in this set consume a resource without taking ownership. Their use
// does not make an otherwise local resource safe. Unknown calls, on the other
// hand, are treated conservatively as a possible ownership transfer.
var unmanagedKnownNonOwningCalls = map[string]struct{}{
	"line-seq":               {},
	"clojure.core/line-seq":  {},
	"read-line":              {},
	"clojure.core/read-line": {},
	"clojure.java.io/copy":   {},
	"io/copy":                {},
	"count":                  {},
	"doall":                  {},
	"dorun":                  {},
	"run!":                   {},
}

var unmanagedStructuralForms = map[string]struct{}{
	"let": {}, "let*": {}, "loop": {}, "loop*": {},
	"if": {}, "if-let": {}, "if-not": {}, "when": {}, "when-let": {}, "when-not": {},
	"do": {}, "try": {}, "catch": {}, "finally": {}, "cond": {}, "case": {},
	"fn": {}, "fn*": {}, "defn": {}, "defn-": {}, "binding": {},
}

func unmanagedNodeIsSymbol(node *reader.RichNode, values ...string) bool {
	if node == nil || node.Type != reader.NodeSymbol {
		return false
	}
	for _, value := range values {
		if node.Value == value {
			return true
		}
	}
	return false
}

func unmanagedCanonicalOperation(node *reader.RichNode, context map[string]interface{}) string {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol {
		return ""
	}
	symbol := node.Children[0].Value
	if symbol == "new" && len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
		className := node.Children[1].Value
		if resolved := node.Children[1].Resolution; resolved != nil &&
			resolved.Kind != reader.ResolutionUnresolved && resolved.Kind != reader.ResolutionLocal {
			className = resolved.CanonicalName
		}
		constructor := strings.TrimSuffix(className, ".") + "."
		if _, ok := unmanagedResourceCreators[constructor]; ok {
			return constructor
		}
		return ""
	}
	if resolved := rules.ResolvedCall(node); resolved != nil &&
		resolved.Kind != reader.ResolutionUnresolved && resolved.Kind != reader.ResolutionLocal {
		if _, ok := unmanagedResourceCreators[resolved.CanonicalName]; ok {
			return resolved.CanonicalName
		}
	}
	if strings.HasSuffix(symbol, ".") {
		if _, ok := unmanagedResourceCreators[symbol]; ok {
			return symbol
		}
	}
	if symbol == ".getConnection" || symbol == "getConnection" ||
		symbol == ".getInputStream" || symbol == "getInputStream" {
		return symbol
	}
	return ""
}

func unmanagedPriorBindingInitializer(node *reader.RichNode, context map[string]interface{}, name string) *reader.RichNode {
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil || parent.Type != reader.NodeVector {
		return nil
	}
	currentIndex := -1
	for index, child := range parent.Children {
		if child == node {
			currentIndex = index
			break
		}
	}
	if currentIndex < 0 {
		return nil
	}
	for index := 0; index+1 < currentIndex; index += 2 {
		if unmanagedNodeIsSymbol(parent.Children[index], name) {
			return parent.Children[index+1]
		}
	}
	return nil
}

func unmanagedWrapsKnownLocalResource(node *reader.RichNode, context map[string]interface{}, operation string) bool {
	if node == nil {
		return false
	}
	for _, argument := range node.Children[1:] {
		if argument.Type != reader.NodeSymbol {
			continue
		}
		initializer := unmanagedPriorBindingInitializer(node, context, argument.Value)
		if initializer == nil {
			continue
		}
		if unmanagedContainsInMemoryResource(initializer) {
			return true
		}
		// reader/writer are adapters over an already-owned local resource. The
		// underlying creator is the lifecycle root and is analyzed separately.
		if operation == "clojure.java.io/reader" || operation == "clojure.java.io/writer" {
			if unmanagedCanonicalOperation(initializer, context) != "" {
				return true
			}
		}
	}
	return false
}

func unmanagedContainsInMemoryResource(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol {
		if node.Value == "System/out" || node.Value == "System/err" || node.Value == "System/in" {
			return true
		}
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
		if _, ok := unmanagedInMemoryCreators[node.Children[0].Value]; ok {
			return true
		}
	}
	for _, child := range node.Children {
		if unmanagedContainsInMemoryResource(child) {
			return true
		}
	}
	return false
}

func unmanagedContainsClose(node *reader.RichNode, binding string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeList && len(node.Children) >= 2 {
		// (.close resource) and (close resource)
		if unmanagedNodeIsSymbol(node.Children[0], ".close", "close") &&
			unmanagedNodeIsSymbol(node.Children[1], binding) {
			return true
		}
		// (. resource close)
		if len(node.Children) >= 3 && unmanagedNodeIsSymbol(node.Children[0], ".") &&
			unmanagedNodeIsSymbol(node.Children[1], binding) &&
			unmanagedNodeIsSymbol(node.Children[2], "close") {
			return true
		}
	}
	for _, child := range node.Children {
		if unmanagedContainsClose(child, binding) {
			return true
		}
	}
	return false
}

func unmanagedBindingInfo(node *reader.RichNode, context map[string]interface{}) (string, *reader.RichNode) {
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil || parent.Type != reader.NodeVector {
		return "", nil
	}

	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	if len(ancestors) < 2 || ancestors[len(ancestors)-1] != parent {
		return "", nil
	}
	bindingForm := ancestors[len(ancestors)-2]
	if bindingForm.Type != reader.NodeList || len(bindingForm.Children) < 2 ||
		bindingForm.Children[1] != parent || bindingForm.Children[0].Type != reader.NodeSymbol {
		return "", nil
	}
	switch bindingForm.Children[0].Value {
	case "let", "let*", "loop", "loop*", "if-let", "when-let", "binding":
	default:
		return "", nil
	}

	for index, child := range parent.Children {
		if child == node && index > 0 && index%2 == 1 && parent.Children[index-1].Type == reader.NodeSymbol {
			return parent.Children[index-1].Value, bindingForm
		}
	}
	return "", nil
}

func unmanagedIsReturnedBinding(node *reader.RichNode, binding string) bool {
	if node == nil {
		return false
	}
	if len(node.Children) < 3 {
		return false
	}
	return unmanagedResultMayReturnBinding(node.Children[len(node.Children)-1], binding)
}

func unmanagedResultMayReturnBinding(node *reader.RichNode, binding string) bool {
	if node == nil {
		return false
	}
	if unmanagedNodeIsSymbol(node, binding) {
		return true
	}
	// A resource placed in a returned map, vector, or set escapes to the
	// caller together with the collection, e.g. `{:server-socket ss}` handed
	// to a matching `stop` function.
	if node.Type == reader.NodeMap || node.Type == reader.NodeVector || node.Type == reader.NodeSet {
		return unmanagedNodeContainsSymbol(node, binding)
	}
	if node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0] == nil ||
		node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	switch node.Children[0].Value {
	case "if", "if-not":
		if len(node.Children) >= 3 && unmanagedResultMayReturnBinding(node.Children[2], binding) {
			return true
		}
		return len(node.Children) >= 4 && unmanagedResultMayReturnBinding(node.Children[3], binding)
	case "when", "when-not", "do", "try":
		return len(node.Children) > 1 && unmanagedResultMayReturnBinding(node.Children[len(node.Children)-1], binding)
	case "let", "let*", "binding", "locking":
		return len(node.Children) > 2 && unmanagedResultMayReturnBinding(node.Children[len(node.Children)-1], binding)
	case "cond":
		for index := 2; index < len(node.Children); index += 2 {
			if unmanagedResultMayReturnBinding(node.Children[index], binding) {
				return true
			}
		}
		return len(node.Children)%2 == 1 && unmanagedResultMayReturnBinding(node.Children[len(node.Children)-1], binding)
	case "case":
		for index := 3; index < len(node.Children); index += 2 {
			if unmanagedResultMayReturnBinding(node.Children[index], binding) {
				return true
			}
		}
		return len(node.Children)%2 == 1 && unmanagedResultMayReturnBinding(node.Children[len(node.Children)-1], binding)
	default:
		return false
	}
}

func unmanagedNodeContainsSymbol(node *reader.RichNode, binding string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol && node.Value == binding {
		return true
	}
	if node.Type == reader.NodeVector || node.Type == reader.NodeSet || node.Type == reader.NodeMap {
		for _, child := range node.Children {
			if unmanagedNodeContainsSymbol(child, binding) {
				return true
			}
		}
	}
	return false
}

func unmanagedCallUsesBinding(node *reader.RichNode, binding string) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 || node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	for _, argument := range node.Children[1:] {
		if unmanagedNodeContainsSymbol(argument, binding) {
			return true
		}
	}
	return false
}

func unmanagedTransferredToUnknownCall(node *reader.RichNode, binding string) bool {
	if node == nil {
		return false
	}
	if unmanagedCallUsesBinding(node, binding) {
		head := node.Children[0].Value
		if head != ".close" && head != "close" && !strings.HasPrefix(head, ".") {
			if _, structural := unmanagedStructuralForms[head]; !structural {
				if _, knownNonOwner := unmanagedKnownNonOwningCalls[head]; !knownNonOwner {
					return true
				}
			}
		}
	}
	for _, child := range node.Children {
		if unmanagedTransferredToUnknownCall(child, binding) {
			return true
		}
	}
	return false
}

func unmanagedHasSafeLifecycle(bindingForm *reader.RichNode, binding string) bool {
	if unmanagedContainsClose(bindingForm, binding) {
		return true
	}
	if unmanagedIsReturnedBinding(bindingForm, binding) {
		return true
	}
	return unmanagedTransferredToUnknownCall(bindingForm, binding)
}

// unmanagedJavaAdapters wrap an existing stream and open nothing themselves.
var unmanagedJavaAdapters = map[string]bool{
	"java.io.BufferedReader.": true, "BufferedReader.": true,
	"java.io.BufferedWriter.": true, "BufferedWriter.": true,
	"java.io.InputStreamReader.": true, "InputStreamReader.": true,
	"java.io.OutputStreamWriter.": true, "OutputStreamWriter.": true,
	"java.io.BufferedInputStream.": true, "BufferedInputStream.": true,
	"java.io.BufferedOutputStream.": true, "BufferedOutputStream.": true,
	"java.util.zip.GZIPInputStream.": true, "GZIPInputStream.": true,
	"java.util.zip.GZIPOutputStream.": true, "GZIPOutputStream.": true,
	"java.util.zip.ZipInputStream.": true, "ZipInputStream.": true,
	"java.util.zip.ZipOutputStream.": true, "ZipOutputStream.": true,
}

// unmanagedDualOpeners open a resource for a path/file/URL argument but only
// adapt an existing stream argument.
var unmanagedDualOpeners = map[string]bool{
	"clojure.java.io/reader": true, "clojure.java.io/writer": true,
	"clojure.java.io/input-stream": true, "clojure.java.io/output-stream": true,
}

func unmanagedAdapterSource(node *reader.RichNode) *reader.RichNode {
	if node == nil || len(node.Children) < 2 {
		return nil
	}
	if unmanagedNodeIsSymbol(node.Children[0], "new") {
		if len(node.Children) < 3 {
			return nil
		}
		return node.Children[2]
	}
	return node.Children[1]
}

// unmanagedFreshSource proves that a source denotes something the call opens
// or a resource created right there, never a stream owned by someone else.
func unmanagedFreshSource(node *reader.RichNode, context map[string]interface{}) bool {
	if node == nil {
		return false
	}
	switch node.Type {
	case reader.NodeString:
		return true
	case reader.NodeList:
		if len(node.Children) == 0 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
			return false
		}
		head := node.Children[0].Value
		// (FileOutputStream. FileDescriptor/out) wraps the process's standard
		// streams, which must never be closed by the wrapper's user.
		for _, arg := range node.Children[1:] {
			if arg != nil && arg.Type == reader.NodeSymbol &&
				(strings.HasPrefix(arg.Value, "FileDescriptor/") || strings.HasPrefix(arg.Value, "java.io.FileDescriptor/")) {
				return false
			}
		}
		if head == "or" || head == "clojure.core/or" {
			for _, branch := range node.Children[1:] {
				if !unmanagedFreshSource(branch, context) {
					return false
				}
			}
			return len(node.Children) > 1
		}
		if head == "if" || head == "if-not" {
			if len(node.Children) < 4 {
				return false
			}
			return unmanagedFreshSource(node.Children[2], context) && unmanagedFreshSource(node.Children[3], context)
		}
		if rules.CallResolvesTo(node, "clojure.java.io/file", "clojure.java.io/resource", "clojure.core/str") ||
			head == "File." || head == "java.io.File." || head == "URL." || head == "java.net.URL." {
			return true
		}
		operation := unmanagedCanonicalOperation(node, context)
		if operation == "" || strings.Contains(operation, "getInputStream") || strings.Contains(operation, "getConnection") {
			return false
		}
		if unmanagedJavaAdapters[operation] || unmanagedDualOpeners[operation] {
			return unmanagedFreshSource(unmanagedAdapterSource(node), context)
		}
		return true
	}
	return false
}

// unmanagedIsStandardStream reports a non-local *in*, *out*, or *err*. The
// symbol is matched by name because silence is the conservative outcome; a
// local binding of the same name is a value of unknown ownership instead.
func unmanagedIsStandardStream(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeSymbol ||
		(node.Resolution != nil && node.Resolution.Kind == reader.ResolutionLocal) {
		return false
	}
	switch strings.TrimPrefix(node.Value, "clojure.core/") {
	case "*in*", "*out*", "*err*":
		return true
	}
	return false
}

// unmanagedOpensThroughDualOpener reports an adapter chain that ends in a
// clojure.java.io opener, e.g. (GZIPInputStream. (io/input-stream x)): the
// chain may open a resource, so the finding is kept (as contextual when x is
// not a proven fresh source).
func unmanagedOpensThroughDualOpener(node *reader.RichNode, context map[string]interface{}) bool {
	operation := unmanagedCanonicalOperation(node, context)
	if unmanagedDualOpeners[operation] {
		return true
	}
	if unmanagedJavaAdapters[operation] {
		return unmanagedOpensThroughDualOpener(unmanagedAdapterSource(node), context)
	}
	return false
}

// unmanagedEscapes reports ownership transfers that the direct-return and
// direct-argument checks miss: the resource placed in a collection literal,
// captured by a closure that may outlive the scope, or reached through a
// same-vector alias (e.g. fc (.getChannel raf)) handed to an unknown call.
func unmanagedEscapes(bindingForm *reader.RichNode, binding string) bool {
	if unmanagedCapturedOrCollected(bindingForm, binding, false) {
		return true
	}
	if len(bindingForm.Children) < 2 || bindingForm.Children[1] == nil || bindingForm.Children[1].Type != reader.NodeVector {
		return false
	}
	vector := bindingForm.Children[1]
	for index := 0; index+1 < len(vector.Children); index += 2 {
		alias := vector.Children[index]
		initializer := vector.Children[index+1]
		if alias == nil || alias.Type != reader.NodeSymbol || alias.Value == binding || alias.Value == "_" ||
			!unmanagedMentions(initializer, binding) {
			continue
		}
		if unmanagedTransferredToUnknownCall(bindingForm, alias.Value) ||
			unmanagedCapturedOrCollected(bindingForm, alias.Value, false) {
			return true
		}
	}
	return false
}

func unmanagedMentions(node *reader.RichNode, symbol string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol && node.Value == symbol {
		return true
	}
	for _, child := range node.Children {
		if unmanagedMentions(child, symbol) {
			return true
		}
	}
	return false
}

func unmanagedCapturedOrCollected(node *reader.RichNode, binding string, insideCapture bool) bool {
	if node == nil {
		return false
	}
	if insideCapture && node.Type == reader.NodeSymbol && node.Value == binding {
		return true
	}
	capture := insideCapture
	switch node.Type {
	case reader.NodeMap, reader.NodeSet, reader.NodeFnLiteral:
		capture = true
	case reader.NodeList:
		if len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
			switch strings.TrimPrefix(node.Children[0].Value, "clojure.core/") {
			case "fn", "fn*", "reify", "proxy", "deftype", "defrecord", "hash-map", "vector", "list", "atom", "agent", "ref", "volatile!":
				capture = true
			}
		}
	}
	for _, child := range node.Children {
		if unmanagedCapturedOrCollected(child, binding, capture) {
			return true
		}
	}
	return false
}

func unmanagedIsNonEvaluated(context map[string]interface{}) bool {
	enclosing, _ := context["enclosingForms"].([]string)
	for _, form := range enclosing {
		switch form {
		case "comment", "quote", "__non-evaluated__":
			return true
		}
	}
	return false
}

func (r *UnmanagedResourceIORule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 ||
		node.Children[0].Type != reader.NodeSymbol {
		return nil
	}
	execution := rules.CurrentExecutionContext(context)
	if inside, _ := context["isInsideWithOpen"].(bool); inside ||
		execution == rules.ExecutionNonEvaluated || execution == rules.ExecutionUnknown ||
		unmanagedIsNonEvaluated(context) {
		return nil
	}

	operation := unmanagedCanonicalOperation(node, context)
	if operation == "" || unmanagedContainsInMemoryResource(node) ||
		unmanagedWrapsKnownLocalResource(node, context, operation) {
		return nil
	}

	// High-precision mode: only report resources assigned to a lexical binding.
	// A resource used as an argument or returned directly may have transferred
	// ownership, and is intentionally left unreported.
	binding, bindingForm := unmanagedBindingInfo(node, context)
	if binding == "" || bindingForm == nil || unmanagedHasSafeLifecycle(bindingForm, binding) ||
		unmanagedEscapes(bindingForm, binding) {
		return nil
	}
	// (binding [*out* (io/writer out)] ...) over a function parameter adapts
	// the caller's stream for printing; the caller owns and closes it.
	if bindingForm.Children[0].Value == "binding" && len(node.Children) > 1 &&
		node.Children[1] != nil && node.Children[1].Type == reader.NodeSymbol &&
		unmanagedIsFunctionParameter(node.Children[1].Value, context) {
		return nil
	}
	// More generally, rebinding *in*/*out*/*err* to an adapter over a stream
	// that is not provably opened here (a process stream, a standard file
	// descriptor) redirects I/O; closing the adapter would close the source.
	if bindingForm.Children[0].Value == "binding" &&
		(unmanagedDualOpeners[operation] || unmanagedJavaAdapters[operation]) &&
		!unmanagedFreshSource(unmanagedAdapterSource(node), context) {
		return nil
	}
	// An adapter or io/writer over *in*, *out*, or *err* wraps the process's
	// standard streams, which the wrapper's user must never close.
	if (unmanagedDualOpeners[operation] || unmanagedJavaAdapters[operation]) &&
		unmanagedIsStandardStream(unmanagedAdapterSource(node)) {
		return nil
	}
	// A Java adapter never opens an OS resource itself: it only owns what it
	// wraps. Wrapping a stream owned elsewhere (a parameter, *in*, a request
	// body) must not be closed here, so only a freshly created source counts.
	if unmanagedJavaAdapters[operation] && !unmanagedFreshSource(unmanagedAdapterSource(node), context) &&
		!unmanagedOpensThroughDualOpener(unmanagedAdapterSource(node), context) {
		return nil
	}

	finding := &rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Resource created by `%s` is bound to `%s` without a proven close; use with-open or close it in finally.",
			operation, binding),
		Filepath:   filepath,
		Location:   node.Location,
		Severity:   rules.ContextualSeverity(context, r.Severity),
		Confidence: rules.ConfidenceProven,
		Tags:       rules.ContextualTags(context),
	}
	bindingName := binding[strings.LastIndex(binding, "/")+1:]
	if strings.HasPrefix(bindingName, "*") || strings.Contains(operation, "getInputStream") || strings.Contains(operation, "getConnection") {
		return rules.SetContextualFindingWithEvidence(finding, "Resource ownership may belong to an external object or lifecycle that cannot be proven locally.", "resource-ownership", "resource-lifecycle")
	}
	// clojure.java.io/reader and friends open a new resource for a path, file,
	// or URL but only adapt an existing stream. With an argument of unknown
	// type the ownership, and therefore the leak, is not proven.
	if (unmanagedDualOpeners[operation] || unmanagedJavaAdapters[operation]) && !unmanagedFreshSource(unmanagedAdapterSource(node), context) {
		return rules.SetContextualFindingWithEvidence(finding, "The argument may be a stream owned by the caller, which this adapter must not close; only a path, file, URL, or newly opened resource proves local ownership.", "resource-ownership")
	}
	return finding
}

func init() {
	defaultRule := &UnmanagedResourceIORule{
		Rule: rules.Rule{
			ID:          "unmanaged-resource-io",
			Name:        "Unmanaged Resource I/O",
			Description: "Detects locally-owned Closeable resources without a proven close operation.",
			Severity:    rules.SeverityWarning,
		},
	}

	rules.RegisterRule(defaultRule)
}

func unmanagedIsFunctionParameter(name string, context map[string]interface{}) bool {
	ancestors, _ := context["ancestorNodes"].([]*reader.RichNode)
	hasParam := func(params *reader.RichNode) bool {
		if params == nil || params.Type != reader.NodeVector {
			return false
		}
		for _, p := range params.Children {
			if p != nil && p.Type == reader.NodeSymbol && p.Value == name {
				return true
			}
		}
		return false
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		form := ancestors[i]
		if form == nil || form.Type != reader.NodeList || len(form.Children) < 2 || form.Children[0] == nil {
			continue
		}
		// Arity list ([params] body...) inside a multi-arity fn/defn.
		if form.Children[0].Type == reader.NodeVector {
			if hasParam(form.Children[0]) {
				return true
			}
			continue
		}
		switch form.Children[0].Value {
		case "fn", "fn*", "defn", "defn-":
			for _, child := range form.Children[1:] {
				if child != nil && child.Type == reader.NodeVector {
					if hasParam(child) {
						return true
					}
					break
				}
			}
		}
	}
	return false
}
