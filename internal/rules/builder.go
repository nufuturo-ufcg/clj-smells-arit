package rules

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
)

// Predicate defines a matching condition on an AST node, context, and filepath.
type Predicate func(node *reader.RichNode, context map[string]interface{}, filepath string) bool

// DSLRule represents a generic rule whose matching criteria and messages are dynamically defined.
type DSLRule struct {
	meta            Rule
	predicates      []Predicate
	msgBuilder      func(node *reader.RichNode, context map[string]interface{}) string
	severityBuilder func(node *reader.RichNode, context map[string]interface{}, defaultSev Severity) Severity
}

// Meta returns the rule metadata.
func (r *DSLRule) Meta() Rule {
	return r.meta
}

// Check executes all associated predicates on the node. If all match, it returns a Finding.
func (r *DSLRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	for _, pred := range r.predicates {
		if !pred(node, context, filepath) {
			return nil
		}
	}
	message := r.msgBuilder(node, context)
	sev := r.meta.Severity
	if r.severityBuilder != nil {
		sev = r.severityBuilder(node, context, sev)
	}
	return &Finding{
		RuleID:   r.meta.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: sev,
	}
}

// Builder provides a fluent API to construct DSLRule instances.
type Builder struct {
	rule DSLRule
}

// NewRule starts the configuration chain for a new rule with the given ID.
func NewRule(id string) *Builder {
	group := "clojure-specific" // default
	for i := 1; i < 6; i++ {
		_, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if strings.Contains(file, "/internal/rules/") && !strings.HasSuffix(file, "builder.go") && !strings.HasSuffix(file, "rules.go") {
			if strings.Contains(file, "/clojurespecific/") {
				group = "clojure-specific"
				break
			} else if strings.Contains(file, "/functional/") {
				group = "functional"
				break
			} else if strings.Contains(file, "/traditional/") {
				group = "traditional"
				break
			}
		}
	}

	return &Builder{
		rule: DSLRule{
			meta: Rule{
				ID:       id,
				Severity: SeverityWarning, // Default
				Group:    group,
			},
		},
	}
}

// Name sets the human-readable name of the rule.
func (b *Builder) Name(name string) *Builder {
	b.rule.meta.Name = name
	return b
}

// Description sets the detailed description of the rule.
func (b *Builder) Description(desc string) *Builder {
	b.rule.meta.Description = desc
	return b
}

// Severity sets the severity of the rule (Warning, Info, Hint).
func (b *Builder) Severity(sev Severity) *Builder {
	b.rule.meta.Severity = sev
	return b
}

// When adds a validation predicate to the rule.
func (b *Builder) When(predicate Predicate) *Builder {
	b.rule.predicates = append(b.rule.predicates, predicate)
	return b
}

// Message sets a static error message for the rule finding.
func (b *Builder) Message(msg string) *Builder {
	b.rule.msgBuilder = func(node *reader.RichNode, context map[string]interface{}) string {
		return msg
	}
	return b
}

// MessageFunc sets a dynamic message builder function for the rule finding.
func (b *Builder) MessageFunc(msgBuilder func(node *reader.RichNode, context map[string]interface{}) string) *Builder {
	b.rule.msgBuilder = msgBuilder
	return b
}

// SeverityFunc sets a dynamic severity builder function for the rule finding.
func (b *Builder) SeverityFunc(sevBuilder func(node *reader.RichNode, context map[string]interface{}, defaultSev Severity) Severity) *Builder {
	b.rule.severityBuilder = sevBuilder
	return b
}

// Register adds the rule to the global registry and returns it.
func (b *Builder) Register() CheckerRule {
	RegisterRule(&b.rule)
	return &b.rule
}

// --- Built-in Predicates ---

// IsList checks if the node is a list.
func IsList() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeList
	}
}

// IsVector checks if the node is a vector.
func IsVector() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeVector
	}
}

// IsMap checks if the node is a map.
func IsMap() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeMap
	}
}

// IsSet checks if the node is a set.
func IsSet() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeSet
	}
}

// IsSymbol checks if the node is a symbol.
func IsSymbol() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeSymbol
	}
}

// IsKeyword checks if the node is a keyword.
func IsKeyword() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeKeyword
	}
}

// IsString checks if the node is a string literal.
func IsString() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeString
	}
}

// IsNumber checks if the node is a number literal.
func IsNumber() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Type == reader.NodeNumber
	}
}

// ValueEquals checks if the text value of the node matches the given string.
func ValueEquals(val string) Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && node.Value == val
	}
}

// FirstChildValueEquals checks if the first child of a list/vector has the given value.
func FirstChildValueEquals(val string) Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Value == val
	}
}

// HasMinChildren checks if the composite node has at least count children.
func HasMinChildren(count int) Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && len(node.Children) >= count
	}
}

// HasChildrenCount checks if the composite node has exactly count children.
func HasChildrenCount(count int) Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		return node != nil && len(node.Children) == count
	}
}

// ChildMatches checks if the child node at the given index satisfies the specified predicate.
func ChildMatches(index int, pred Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		if node == nil || index < 0 || index >= len(node.Children) {
			return false
		}
		return pred(node.Children[index], context, filepath)
	}
}

// Not negates the result of a predicate.
func Not(pred Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		return !pred(node, context, filepath)
	}
}

// Any checks if at least one of the provided predicates is true.
func Any(preds ...Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		for _, pred := range preds {
			if pred(node, context, filepath) {
				return true
			}
		}
		return false
	}
}

// All checks if all of the provided predicates are true.
func All(preds ...Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		for _, pred := range preds {
			if !pred(node, context, filepath) {
				return false
			}
		}
		return true
	}
}

// FilepathContains checks if the analyzed filepath contains any of the provided substrings.
func FilepathContains(substrs ...string) Predicate {
	return func(_ *reader.RichNode, _ map[string]interface{}, filepath string) bool {
		for _, sub := range substrs {
			if strings.Contains(filepath, sub) {
				return true
			}
		}
		return false
	}
}

// IsInside checks if the node is nested under any of the specified form names in the context hierarchy.
func IsInside(formNames ...string) Predicate {
	return func(_ *reader.RichNode, context map[string]interface{}, _ string) bool {
		return isInsideContext(context, formNames)
	}
}

// IsLocalScope checks if the execution occurs in a local lexical scope (function, let, loop, or binding).
func IsLocalScope() Predicate {
	return func(_ *reader.RichNode, context map[string]interface{}, _ string) bool {
		scopes := []string{"isInsideFunction", "isInsideLet", "isInsideLoop", "isInsideBinding"}
		for _, scope := range scopes {
			if val, ok := context[scope].(bool); ok && val {
				return true
			}
		}
		return false
	}
}

// ToClojureString reconstructs a Clojure code representation from the AST node.
func ToClojureString(node *reader.RichNode) string {
	if node == nil {
		return ""
	}
	if node.Value != "" {
		return node.Value
	}
	var childrenStr []string
	for _, child := range node.Children {
		childrenStr = append(childrenStr, ToClojureString(child))
	}
	joined := strings.Join(childrenStr, " ")
	switch node.Type {
	case reader.NodeList:
		return "(" + joined + ")"
	case reader.NodeVector:
		return "[" + joined + "]"
	case reader.NodeMap:
		return "{" + joined + "}"
	case reader.NodeSet:
		return "#{" + joined + "}"
	}
	return ""
}

// GetConfigInt retrieves an integer setting for the rule from the context.
func GetConfigInt(context map[string]interface{}, ruleID string, key string, defaultValue int) int {
	if cfgVal, ok := context["config"]; ok {
		if cfg, ok := cfgVal.(*config.Config); ok && cfg != nil {
			return cfg.GetRuleSettingInt(ruleID, key, defaultValue)
		}
	}
	return defaultValue
}

// GetConfigBool retrieves a boolean setting for the rule from the context.
func GetConfigBool(context map[string]interface{}, ruleID string, key string, defaultValue bool) bool {
	if cfgVal, ok := context["config"]; ok {
		if cfg, ok := cfgVal.(*config.Config); ok && cfg != nil {
			return cfg.GetRuleSettingBool(ruleID, key, defaultValue)
		}
	}
	return defaultValue
}

// GetConfigStringSlice retrieves a slice of strings setting for the rule from the context.
func GetConfigStringSlice(context map[string]interface{}, ruleID string, key string) []string {
	if cfgVal, ok := context["config"]; ok {
		if cfg, ok := cfgVal.(*config.Config); ok && cfg != nil {
			if ruleSettings, ok := cfg.RuleConfig[ruleID]; ok {
				if value, ok := ruleSettings[key]; ok {
					if slice, ok := value.([]interface{}); ok {
						var res []string
						for _, item := range slice {
							if str, ok := item.(string); ok {
								res = append(res, str)
							}
						}
						return res
					} else if slice, ok := value.([]string); ok {
						return slice
					}
				}
			}
		}
	}
	return nil
}

// HasDescendant checks recursively if any descendant of the current node satisfies the predicate.
func HasDescendant(pred Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		if node == nil {
			return false
		}
		var walk func(n *reader.RichNode) bool
		walk = func(n *reader.RichNode) bool {
			if n == nil {
				return false
			}
			if pred(n, context, filepath) {
				return true
			}
			for _, child := range n.Children {
				if walk(child) {
					return true
				}
			}
			return false
		}
		for _, child := range node.Children {
			if walk(child) {
				return true
			}
		}
		return false
	}
}

// ChildValueEquals is a shortcut that checks if a child at the given index has a specific value.
func ChildValueEquals(index int, val string) Predicate {
	return ChildMatches(index, ValueEquals(val))
}

// ChildIsSymbol is a shortcut that checks if a child at the given index is a symbol.
func ChildIsSymbol(index int) Predicate {
	return ChildMatches(index, IsSymbol())
}

// HasBindingPair checks if a binding vector (e.g. in let/loop) contains a key-value pair matching the predicates.
func HasBindingPair(keyPred, valPred Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		if node == nil || node.Type != reader.NodeVector {
			return false
		}
		for i := 0; i+1 < len(node.Children); i += 2 {
			if keyPred(node.Children[i], context, filepath) && valPred(node.Children[i+1], context, filepath) {
				return true
			}
		}
		return false
	}
}

// IsResolvedTo checks if the node's resolved symbol matches the specified type name (e.g., "parameter", "variable").
func IsResolvedTo(symType string) Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		if node == nil || node.SymbolRef == nil {
			return false
		}
		val := reflect.ValueOf(node.SymbolRef)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			return false
		}
		typeField := val.FieldByName("Type")
		if !typeField.IsValid() {
			return false
		}
		return typeField.String() == symType
	}
}

// IsResolvedPrivate checks if the node resolves to a private symbol.
func IsResolvedPrivate() Predicate {
	return func(node *reader.RichNode, _ map[string]interface{}, _ string) bool {
		if node == nil || node.SymbolRef == nil {
			return false
		}
		val := reflect.ValueOf(node.SymbolRef)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			return false
		}
		pField := val.FieldByName("IsPrivate")
		if !pField.IsValid() {
			return false
		}
		return pField.Bool()
	}
}

// AnyChildMatches checks if at least one direct child of the node satisfies the predicate.
func AnyChildMatches(pred Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		if node == nil {
			return false
		}
		for _, child := range node.Children {
			if pred(child, context, filepath) {
				return true
			}
		}
		return false
	}
}

// AllChildrenMatch checks if all direct children of the node satisfy the predicate.
func AllChildrenMatch(pred Predicate) Predicate {
	return func(node *reader.RichNode, context map[string]interface{}, filepath string) bool {
		if node == nil || len(node.Children) == 0 {
			return false
		}
		for _, child := range node.Children {
			if !pred(child, context, filepath) {
				return false
			}
		}
		return true
	}
}
