package analyzer

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// ExpandMacros traverses a derived AST and expands only deterministic,
// locally representable threading forms. It is strictly experimental.
func ExpandMacros(roots []*reader.RichNode) {
	for _, root := range roots {
		expandNodeRecursively(root)
	}
}

func expandNodeRecursively(node *reader.RichNode) {
	if node == nil {
		return
	}
	switch node.Type {
	case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
		return
	}
	for _, child := range node.Children {
		expandNodeRecursively(child)
	}
	if node.Type != reader.NodeList || len(node.Children) < 3 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
		return
	}
	switch node.Children[0].Value {
	case "->":
		expandThread(node, false)
	case "->>":
		expandThread(node, true)
	case "some->":
		expandSomeThread(node, false)
	case "some->>":
		expandSomeThread(node, true)
	case "cond->":
		expandConditionalThread(node, false)
	case "cond->>":
		expandConditionalThread(node, true)
	}
}

func expandThread(node *reader.RichNode, last bool) {
	current := node.Children[1]
	macro := node.Children[0].Value
	for _, form := range node.Children[2:] {
		current = threadedCall(form, current, last, macro)
	}
	replaceWithDerived(node, current, macro)
}

// some-> uses nil? rather than truthiness: false is a valid value and must
// continue through the pipeline. Each intermediate value is bound once.
func expandSomeThread(node *reader.RichNode, last bool) {
	current := node.Children[1]
	macro := node.Children[0].Value
	for index, form := range node.Children[2:] {
		name := freshGeneratedName(node, macro, index)
		binding := generatedSymbol(name, node.Location, macro)
		value := threadedCall(form, binding, last, macro)
		nilTest := generatedList(node.Location, macro,
			generatedSymbol("nil?", node.Location, macro), binding)
		conditional := generatedList(node.Location, macro,
			nilTest, generatedNil(node.Location, macro), value)
		current = generatedList(node.Location, macro,
			generatedSymbol("let", node.Location, macro),
			generatedVector(node.Location, macro, binding, current), conditional)
	}
	replaceWithDerived(node, current, macro)
}

// cond-> and cond->> evaluate the current value once per stage, apply a form
// only when its test is truthy, and carry the previous value forward. Invalid
// odd arity is left untouched because no safe local expansion exists.
func expandConditionalThread(node *reader.RichNode, last bool) {
	if len(node.Children) < 4 || (len(node.Children)-2)%2 != 0 {
		return
	}
	current := node.Children[1]
	macro := node.Children[0].Value
	for pair, index := 0, 2; index < len(node.Children); pair, index = pair+1, index+2 {
		name := freshGeneratedName(node, macro, pair)
		binding := generatedSymbol(name, node.Location, macro)
		value := threadedCall(node.Children[index+1], binding, last, macro)
		conditional := generatedList(node.Location, macro, node.Children[index], value, binding)
		current = generatedList(node.Location, macro,
			generatedSymbol("let", node.Location, macro),
			generatedVector(node.Location, macro, binding, current), conditional)
	}
	replaceWithDerived(node, current, macro)
}

func threadedCall(form, value *reader.RichNode, last bool, macro string) *reader.RichNode {
	if form == nil {
		return generatedList(nil, macro, generatedNil(nil, macro), value)
	}
	if form.Type == reader.NodeList && len(form.Children) > 0 {
		children := make([]*reader.RichNode, 0, len(form.Children)+1)
		if last {
			children = append(children, form.Children...)
			children = append(children, value)
		} else {
			children = append(children, form.Children[0], value)
			children = append(children, form.Children[1:]...)
		}
		return generatedList(form.Location, macro, children...)
	}
	return generatedList(form.Location, macro, form, value)
}

func replaceWithDerived(node, derived *reader.RichNode, macro string) {
	if node == nil || derived == nil {
		return
	}
	originalLocation := node.Location
	node.Type = derived.Type
	node.Value = derived.Value
	node.Children = derived.Children
	node.Metadata = derived.Metadata
	node.Comments = derived.Comments
	node.TypeHint = derived.TypeHint
	node.InferredType = derived.InferredType
	node.Generated = true
	node.Origin = originalLocation
	node.GeneratedBy = macro
	node.Resolution = nil
	node.ResolvedDefinition = nil
	node.Scope = nil
	node.SymbolRef = nil
}

func generatedList(origin *reader.Location, macro string, children ...*reader.RichNode) *reader.RichNode {
	return &reader.RichNode{Type: reader.NodeList, InferredType: "List", Location: origin,
		Generated: true, Origin: origin, GeneratedBy: macro, Children: children}
}

func generatedVector(origin *reader.Location, macro string, children ...*reader.RichNode) *reader.RichNode {
	return &reader.RichNode{Type: reader.NodeVector, Location: origin, Generated: true,
		Origin: origin, GeneratedBy: macro, Children: children}
}

func generatedSymbol(value string, origin *reader.Location, macro string) *reader.RichNode {
	return &reader.RichNode{Type: reader.NodeSymbol, Value: value, Location: origin,
		Generated: true, Origin: origin, GeneratedBy: macro}
}

func generatedNil(origin *reader.Location, macro string) *reader.RichNode {
	return &reader.RichNode{Type: reader.NodeNil, Value: "nil", Location: origin,
		Generated: true, Origin: origin, GeneratedBy: macro}
}

func freshGeneratedName(node *reader.RichNode, macro string, index int) string {
	base := fmt.Sprintf("__arit_%s_%d", strings.ReplaceAll(macro, "-", "_"), index)
	name := base
	for suffix := 1; containsSymbol(node, name); suffix++ {
		name = fmt.Sprintf("%s_%d", base, suffix)
	}
	return name
}

func containsSymbol(node *reader.RichNode, name string) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeSymbol && node.Value == name {
		return true
	}
	for _, child := range node.Children {
		if containsSymbol(child, name) {
			return true
		}
	}
	return containsSymbol(node.Metadata, name)
}
