package rules

import (
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
	return &Builder{
		rule: DSLRule{
			meta: Rule{
				ID:       id,
				Severity: SeverityWarning, // Default
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
		enclosingForms, ok := context["enclosingForms"].([]string)
		if !ok {
			return false
		}
		for _, enclosing := range enclosingForms {
			for _, target := range formNames {
				if enclosing == target {
					return true
				}
			}
		}
		return false
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
