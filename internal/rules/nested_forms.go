package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type NestedFormsRule struct {
	Rule
	MaxConsecutiveSameForms int      `json:"max_consecutive_same_forms" yaml:"max_consecutive_same_forms"`
	MaxConditionalDepth     int      `json:"max_conditional_depth" yaml:"max_conditional_depth"`
	TrackedForms            []string `json:"tracked_forms" yaml:"tracked_forms"`
}

type NestingPattern struct {
	Forms       []string
	Depth       int
	PatternType string
	Nodes       []*reader.RichNode
}

func (r *NestedFormsRule) Meta() Rule {
	return r.Rule
}

func (r *NestedFormsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if !r.isTrackedForm(node) {
		return nil
	}

	pattern := r.buildNestingPattern(node, context)

	if r.isProblematicPattern(pattern) {
		suggestion := r.getSuggestionForPattern(pattern)

		message := fmt.Sprintf(
			"Excessive nesting detected (depth: %d, forms: %s). %s",
			pattern.Depth,
			strings.Join(pattern.Forms, " → "),
			suggestion,
		)

		return &Finding{
			RuleID:   r.ID,
			Message:  message,
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *NestedFormsRule) buildNestingPattern(node *reader.RichNode, context map[string]interface{}) *NestingPattern {
	pattern := &NestingPattern{
		Forms: []string{},
		Depth: 0,
		Nodes: []*reader.RichNode{},
	}

	r.collectNestedForms(node, pattern, 0)
	pattern.PatternType = r.classifyPattern(pattern.Forms)

	return pattern
}

func (r *NestedFormsRule) collectNestedForms(node *reader.RichNode, pattern *NestingPattern, depth int) {
	if !r.isTrackedForm(node) {
		return
	}

	formName := r.getFormName(node)
	pattern.Forms = append(pattern.Forms, formName)
	pattern.Nodes = append(pattern.Nodes, node)
	pattern.Depth = depth + 1

	for _, child := range node.Children {
		if r.isTrackedForm(child) {
			r.collectNestedForms(child, pattern, depth+1)
			return
		}

		for _, grandchild := range child.Children {
			if r.isTrackedForm(grandchild) {
				r.collectNestedForms(grandchild, pattern, depth+1)
				return
			}
		}
	}
}

func (r *NestedFormsRule) isProblematicPattern(pattern *NestingPattern) bool {
	if len(pattern.Forms) < 2 {
		return false
	}

	if r.hasConsecutiveSameForms(pattern.Forms, "let", r.MaxConsecutiveSameForms) {
		return true
	}

	if r.hasConsecutiveSameForms(pattern.Forms, "doseq", 2) {
		return true
	}

	if r.hasConsecutiveSameForms(pattern.Forms, "for", 2) {
		return true
	}

	if r.hasSpecificPattern(pattern.Forms, []string{"let", "when", "let"}) {
		return true
	}

	if r.hasSpecificPattern(pattern.Forms, []string{"let", "if", "let"}) {
		return true
	}

	if r.hasDeepConditionalNesting(pattern.Forms, r.MaxConditionalDepth) {
		return true
	}

	if r.hasConsecutiveSameForms(pattern.Forms, "when-let", 2) {
		return true
	}

	if r.isAcceptablePattern(pattern.Forms) {
		return false
	}

	return false
}

func (r *NestedFormsRule) hasConsecutiveSameForms(forms []string, formName string, minCount int) bool {
	consecutiveCount := 0
	for _, form := range forms {
		if form == formName {
			consecutiveCount++
			if consecutiveCount >= minCount {
				return true
			}
		} else {
			consecutiveCount = 0
		}
	}
	return false
}

func (r *NestedFormsRule) hasSpecificPattern(forms []string, pattern []string) bool {
	if len(forms) < len(pattern) {
		return false
	}

	for i := 0; i <= len(forms)-len(pattern); i++ {
		match := true
		for j, p := range pattern {
			if forms[i+j] != p {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (r *NestedFormsRule) hasDeepConditionalNesting(forms []string, maxDepth int) bool {
	conditionals := map[string]bool{
		"if": true, "when": true, "if-not": true, "when-not": true,
		"if-let": true, "when-let": true, "if-some": true, "when-some": true,
		"cond": true, "case": true,
	}

	conditionalCount := 0
	for _, form := range forms {
		if conditionals[form] {
			conditionalCount++
		}
	}

	return conditionalCount >= maxDepth
}

func (r *NestedFormsRule) isAcceptablePattern(forms []string) bool {

	acceptablePatterns := [][]string{
		{"let", "if"},
		{"let", "when"},
		{"let", "when-not"},
		{"loop", "let"},
		{"binding", "let"},
		{"with-open", "let"},
		{"try", "let"},
		{"let", "try"},
		{"doseq", "when"},
		{"doseq", "if"},
		{"dotimes", "when"},
		{"dotimes", "if"},
	}

	for _, pattern := range acceptablePatterns {
		if r.matchesExactPattern(forms, pattern) {
			return true
		}
	}

	return false
}

func (r *NestedFormsRule) matchesExactPattern(forms []string, pattern []string) bool {
	if len(forms) != len(pattern) {
		return false
	}

	for i, form := range forms {
		if form != pattern[i] {
			return false
		}
	}
	return true
}

func (r *NestedFormsRule) classifyPattern(forms []string) string {
	if r.hasConsecutiveSameForms(forms, "let", 2) {
		return "consecutive-let"
	}
	if r.hasConsecutiveSameForms(forms, "when-let", 2) {
		return "consecutive-when-let"
	}
	if r.hasConsecutiveSameForms(forms, "doseq", 2) {
		return "nested-iteration"
	}
	if r.hasSpecificPattern(forms, []string{"let", "when", "let"}) {
		return "let-when-let"
	}
	return "other"
}

func (r *NestedFormsRule) getSuggestionForPattern(pattern *NestingPattern) string {
	switch pattern.PatternType {
	case "consecutive-let":
		return "Combine multiple 'let' bindings into a single form: (let [x 1, y 2] ...)"

	case "consecutive-when-let":
		return "Consider using 'some->' threading macro: (some-> x f1 f2 f3)"

	case "nested-iteration":
		return "Combine multiple 'doseq' into single form: (doseq [x xs, y ys] ...)"

	case "let-when-let":
		return "Consider using 'when-let' or 'some->' to flatten the nested structure"

	default:
		forms := pattern.Forms
		switch {
		case r.hasSpecificPattern(forms, []string{"let", "if", "let"}):
			return "Consider using 'if-let' or restructuring to avoid nested let forms"

		case r.hasDeepConditionalNesting(forms, 3):
			return "Consider using 'cond' to flatten nested conditional forms"

		case r.hasConsecutiveSameForms(forms, "for", 2):
			return "Combine multiple 'for' comprehensions: (for [x xs, y ys] ...)"

		default:
			return "Consider refactoring to reduce nesting complexity. Use 'some->', 'when-let', or combine forms where possible"
		}
	}
}

func (r *NestedFormsRule) isTrackedForm(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	formName := firstChild.Value
	for _, tracked := range r.TrackedForms {
		if formName == tracked {
			return true
		}
	}

	return false
}

func (r *NestedFormsRule) getFormName(node *reader.RichNode) string {
	if node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol {
		return node.Children[0].Value
	}
	return "unknown"
}

func init() {
	defaultRule := &NestedFormsRule{
		Rule: Rule{
			ID:          "nested-forms",
			Name:        "Nested Forms",
			Description: "Detects problematic nesting patterns like multiple consecutive let forms or unnecessary nested binding forms. This smell occurs when multiple binding or iteration forms are unnecessarily nested instead of being combined in a single, flat form, making code harder to read and reason about.",
			Severity:    SeverityWarning,
		},
		MaxConsecutiveSameForms: 2,
		MaxConditionalDepth:     3,
		TrackedForms: []string{
			"let", "when", "if", "when-let", "if-let", "when-some", "if-some",
			"when-not", "if-not", "loop", "binding", "with-open", "with-local-vars",
			"doseq", "dotimes", "for", "try", "catch", "cond", "case",
		},
	}

	RegisterRule(defaultRule)
}
