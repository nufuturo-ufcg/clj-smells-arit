package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type NestedFormsRule struct {
	Rule
	MaxDepth       int      `json:"max_depth" yaml:"max_depth"`
	TrackedForms   []string `json:"tracked_forms" yaml:"tracked_forms"`
	MinFormsInPath int      `json:"min_forms_in_path" yaml:"min_forms_in_path"`
}

type NestingPath struct {
	Forms []string
	Depth int
	Nodes []*reader.RichNode
}

func (r *NestedFormsRule) Meta() Rule {
	return r.Rule
}

func (r *NestedFormsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if r.isTrackedForm(node) {

		path := r.buildNestingPath(node, context)

		if r.exceedsLimits(path) {
			suggestion := r.getSuggestionForPath(path)

			message := fmt.Sprintf(
				"Excessive nesting detected (depth: %d, forms: %s). %s",
				path.Depth,
				strings.Join(path.Forms, " → "),
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
	}

	return nil
}

func (r *NestedFormsRule) buildNestingPath(node *reader.RichNode, context map[string]interface{}) *NestingPath {
	path := &NestingPath{
		Forms: []string{},
		Depth: 0,
		Nodes: []*reader.RichNode{},
	}

	r.buildPathRecursively(node, context, path)

	return path
}

func (r *NestedFormsRule) buildPathRecursively(node *reader.RichNode, context map[string]interface{}, path *NestingPath) {

	if r.isTrackedForm(node) {
		formName := r.getFormName(node)

		path.Forms = append([]string{formName}, path.Forms...)
		path.Nodes = append([]*reader.RichNode{node}, path.Nodes...)
		path.Depth++
	}

	if parent, ok := context["parent"]; ok {
		if parentNode, ok := parent.(*reader.RichNode); ok {

			parentContext := map[string]interface{}{}
			r.buildPathRecursively(parentNode, parentContext, path)
		}
	}
}

func (r *NestedFormsRule) buildPathFromContext(path *NestingPath, context map[string]interface{}) {

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

func (r *NestedFormsRule) exceedsLimits(path *NestingPath) bool {
	if path.Depth <= 1 {
		return false
	}

	if path.Depth > r.MaxDepth {
		return true
	}

	if len(path.Forms) >= r.MinFormsInPath {
		return true
	}

	return false
}

func (r *NestedFormsRule) getSuggestionForPath(path *NestingPath) string {
	if len(path.Forms) == 0 {
		return "Consider flattening the nested structure."
	}

	switch {
	case r.hasPattern(path.Forms, []string{"let", "when", "let"}):
		return "Consider using 'when-let' or 'some->' threading macro to flatten nested let/when forms."
	case r.hasPattern(path.Forms, []string{"let", "if", "let"}):
		return "Consider using 'if-let' or 'some->' threading macro to flatten nested let/if forms."
	case r.countOccurrences(path.Forms, "let") >= 3:
		return "Consider combining multiple 'let' bindings into a single form or using '->' threading macro."
	case r.countOccurrences(path.Forms, "when") >= 2 && r.countOccurrences(path.Forms, "let") >= 1:
		return "Consider using 'when-let', 'some->', or 'and' to flatten nested when/let conditions."
	case r.hasOnlyConditionals(path.Forms):
		return "Consider using 'cond' to flatten nested conditional forms."
	default:
		return fmt.Sprintf("Consider refactoring to reduce nesting depth. Use threading macros (-> or ->>) or combine forms where possible.")
	}
}

func (r *NestedFormsRule) hasPattern(forms []string, pattern []string) bool {
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

func (r *NestedFormsRule) countOccurrences(forms []string, form string) int {
	count := 0
	for _, f := range forms {
		if f == form {
			count++
		}
	}
	return count
}

func (r *NestedFormsRule) hasOnlyConditionals(forms []string) bool {
	conditionals := map[string]bool{
		"if": true, "when": true, "if-not": true, "when-not": true,
		"if-let": true, "when-let": true, "if-some": true, "when-some": true,
	}

	for _, form := range forms {
		if !conditionals[form] {
			return false
		}
	}
	return len(forms) >= 2
}

func init() {
	defaultRule := &NestedFormsRule{
		Rule: Rule{
			ID:          "nested-forms",
			Name:        "Nested Forms",
			Description: "Detects excessive nesting of forms like let, when, if. Deep nesting makes code harder to read and understand. Consider using threading macros, combining forms, or other refactoring techniques to flatten the structure.",
			Severity:    SeverityWarning,
		},
		MaxDepth:       3,
		MinFormsInPath: 2,
		TrackedForms: []string{
			"let", "when", "if", "when-let", "if-let", "when-some", "if-some",
			"when-not", "if-not", "loop", "binding", "with-open", "with-local-vars",
			"doseq", "dotimes", "for",
		},
	}

	RegisterRule(defaultRule)
}
