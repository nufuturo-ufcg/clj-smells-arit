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

	"java.net.Socket.":       {},
	"Socket.":                {},
	"java.net.ServerSocket.": {},
	"ServerSocket.":          {},
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
	if len(node.Children) > 0 && unmanagedNodeIsSymbol(node.Children[len(node.Children)-1], binding) {
		return true
	}
	return false
}

func unmanagedCallUsesBinding(node *reader.RichNode, binding string) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 || node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	for _, argument := range node.Children[1:] {
		if unmanagedNodeIsSymbol(argument, binding) {
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
		if head == ".close" || head == "close" || strings.HasPrefix(head, ".") {
			return false
		}
		if _, structural := unmanagedStructuralForms[head]; structural {
			return false
		}
		if _, knownNonOwner := unmanagedKnownNonOwningCalls[head]; !knownNonOwner {
			return true
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
	if binding == "" || bindingForm == nil || unmanagedHasSafeLifecycle(bindingForm, binding) {
		return nil
	}

	return &rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Resource created by `%s` is bound to `%s` without a proven close; use with-open or close it in finally.",
			operation, binding),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
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
