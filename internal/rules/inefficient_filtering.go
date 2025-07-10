package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type InefficientFilteringRule struct{}

type PatternDetector struct {
	Name        string
	Description string
	Suggestion  string
	Detector    func(node *reader.RichNode) bool
}

func (r *InefficientFilteringRule) Meta() Rule {
	return Rule{
		ID:          "inefficient-filtering",
		Name:        "Inefficient Filtering",
		Description: "Detects inefficient filtering patterns like `(first (filter ...))`, `(count (filter ...))`, `(empty? (filter ...))`, `(not-empty (filter ...))`, `(map ... (filter ...))`, `(take n (filter ...))`, and multiple consecutive filters. These patterns often process collections unnecessarily or could be optimized with better approaches.",
		Severity:    SeverityHint,
	}
}

func (r *InefficientFilteringRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return nil
	}

	processedPatterns := r.getOrInitProcessedPatterns(context)

	patternKey := r.createPatternKey(node, filepath)
	if processedPatterns[patternKey] {
		return nil
	}

	checks := []func(*reader.RichNode, string, map[string]bool) *Finding{
		r.checkLetBindingPatterns,
		r.checkBasicFilterPatterns,
		r.checkThreadingMacroPatterns,
	}

	for _, check := range checks {
		if finding := check(node, filepath, processedPatterns); finding != nil {
			processedPatterns[patternKey] = true
			return finding
		}
	}

	return nil
}

func (r *InefficientFilteringRule) getOrInitProcessedPatterns(context map[string]interface{}) map[string]bool {
	if context["inefficient_filtering_processed"] == nil {
		context["inefficient_filtering_processed"] = make(map[string]bool)
	}
	return context["inefficient_filtering_processed"].(map[string]bool)
}

func (r *InefficientFilteringRule) createPatternKey(node *reader.RichNode, filepath string) string {
	return fmt.Sprintf("%s:%d:%d", filepath, node.Location.StartLine, node.Location.StartColumn)
}

func (r *InefficientFilteringRule) checkLetBindingPatterns(node *reader.RichNode, filepath string, processedPatterns map[string]bool) *Finding {
	if !r.isLetForm(node) {
		return nil
	}

	bindingsNode := node.Children[1]
	if bindingsNode.Type != reader.NodeVector || len(bindingsNode.Children) < 2 {
		return nil
	}

	for i := 1; i < len(bindingsNode.Children); i += 2 {
		valueNode := bindingsNode.Children[i]

		if finding := r.checkFirstFilterInBinding(valueNode, filepath, processedPatterns); finding != nil {
			return finding
		}
	}

	return nil
}

func (r *InefficientFilteringRule) checkBasicFilterPatterns(node *reader.RichNode, filepath string, processedPatterns map[string]bool) *Finding {
	patterns := []struct {
		name       string
		checker    func(*reader.RichNode) bool
		message    string
		suggestion string
	}{
		{
			name:       "first-filter",
			checker:    r.isFirstFilter,
			message:    "Using `(first (filter ...))` is inefficient.",
			suggestion: "Consider using `(some <predicate> <collection>)` which stops after the first match.",
		},
		{
			name:       "last-filter",
			checker:    r.isLastFilter,
			message:    "Using `(last (filter ...))` is inefficient.",
			suggestion: "This processes the entire collection. Consider reversing the collection first or alternative approaches.",
		},
		{
			name:       "second-filter",
			checker:    r.isSecondFilter,
			message:    "Using `(second (filter ...))` is inefficient.",
			suggestion: "Consider using transducers `(sequence (comp (filter ...) (drop 1) (take 1)) coll)` or `(some ...)` with custom logic.",
		},
		{
			name:       "nth-filter",
			checker:    r.isNthFilter,
			message:    "Using `(nth (filter ...))` is inefficient.",
			suggestion: "Consider using transducers `(sequence (comp (filter ...) (drop n) (take 1)) coll)` for better performance.",
		},
		{
			name:       "count-filter",
			checker:    r.isCountFilter,
			message:    "Using `(count (filter ...))` creates an intermediate collection just to count.",
			suggestion: "Consider using `reduce` with a counter or `transduce` for better performance.",
		},
		{
			name:       "empty-filter",
			checker:    r.isEmptyFilter,
			message:    "Using `(empty? (filter ...))` processes the entire collection.",
			suggestion: "Consider using `(not (some <predicate> <collection>))` which stops early.",
		},
		{
			name:       "not-empty-filter",
			checker:    r.isNotEmptyFilter,
			message:    "Using `(not-empty (filter ...))` processes the entire collection.",
			suggestion: "Consider using `(some <predicate> <collection>)` which returns the first truthy result or nil.",
		},
		{
			name:       "seq-filter",
			checker:    r.isSeqFilter,
			message:    "Using `(seq (filter ...))` is inefficient.",
			suggestion: "Consider using `(some <predicate> <collection>)` which stops after the first match.",
		},
		{
			name:       "nil-seq-filter",
			checker:    r.isNilSeqFilter,
			message:    "Using `(nil? (seq (filter ...)))` is inefficient.",
			suggestion: "Consider using `(not-any? <predicate> <collection>)` which stops early.",
		},
		{
			name:       "not-seq-filter",
			checker:    r.isNotSeqFilter,
			message:    "Using `(not (seq (filter ...)))` is inefficient.",
			suggestion: "Consider using `(not-any? <predicate> <collection>)` which stops early.",
		},
		{
			name:       "map-filter",
			checker:    r.isMapFilter,
			message:    "Using `(map ... (filter ...))` creates an intermediate collection.",
			suggestion: "Consider using `(keep ...)` which combines filtering and mapping in one pass.",
		},
		{
			name:       "take-filter",
			checker:    r.isTakeFilter,
			message:    "Using `(take n (filter ...))` filters the entire collection before taking.",
			suggestion: "Consider using transducers `(sequence (comp (filter ...) (take n)) coll)` for better performance.",
		},
		{
			name:       "distinct-filter",
			checker:    r.isDistinctFilter,
			message:    "Using `(distinct (filter ...))` creates an intermediate collection.",
			suggestion: "Consider using transducers `(sequence (comp (filter ...) (distinct)) coll)` for better performance.",
		},
		{
			name:       "sort-filter",
			checker:    r.isSortFilter,
			message:    "Using `(sort (filter ...))` creates an intermediate collection.",
			suggestion: "Consider using transducers `(sequence (comp (filter ...) (sort)) coll)` for better performance.",
		},
		{
			name:       "reverse-filter",
			checker:    r.isReverseFilter,
			message:    "Using `(reverse (filter ...))` creates an intermediate collection.",
			suggestion: "Consider using `(into () (filter <predicate>) coll)` or transducers for better performance.",
		},
		{
			name:       "doall-filter",
			checker:    r.isDoallFilter,
			message:    "Using `(doall (filter ...))` forces realization and may be unnecessary.",
			suggestion: "Consider if forcing realization is needed, or use transducers for better performance.",
		},
	}

	for _, pattern := range patterns {
		if pattern.checker(node) {

			specificKey := fmt.Sprintf("%s:%s", r.createPatternKey(node, filepath), pattern.name)
			if processedPatterns[specificKey] {
				continue
			}

			meta := r.Meta()
			return &Finding{
				RuleID:   meta.ID,
				Message:  fmt.Sprintf("%s %s", pattern.message, pattern.suggestion),
				Filepath: filepath,
				Location: node.Location,
				Severity: meta.Severity,
			}
		}
	}

	return nil
}

func (r *InefficientFilteringRule) checkThreadingMacroPatterns(node *reader.RichNode, filepath string, _ map[string]bool) *Finding {
	if !r.isThreadingMacro(node) {
		return nil
	}

	filterOps := r.extractFilterOperations(node)
	if len(filterOps) < 2 {
		return nil
	}

	meta := r.Meta()

	if r.hasConsecutiveFilters(filterOps) {
		return &Finding{
			RuleID:   meta.ID,
			Message:  "Multiple consecutive filter operations detected in threading macro. Consider combining predicates into a single filter for better performance: `(filter #(and (pred1 %) (pred2 %)) coll)`",
			Filepath: filepath,
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	if r.hasFilterRemovePattern(filterOps) {
		return &Finding{
			RuleID:   meta.ID,
			Message:  "Filter followed by remove (or vice versa) creates two passes through the collection. Consider combining into a single filter with a compound predicate for better performance.",
			Filepath: filepath,
			Location: node.Location,
			Severity: meta.Severity,
		}
	}

	return nil
}

func (r *InefficientFilteringRule) isLetForm(node *reader.RichNode) bool {
	return len(node.Children) >= 3 &&
		node.Children[0].Type == reader.NodeSymbol &&
		node.Children[0].Value == "let"
}

func (r *InefficientFilteringRule) checkFirstFilterInBinding(valueNode *reader.RichNode, filepath string, processedPatterns map[string]bool) *Finding {
	if !r.isFirstFilter(valueNode) {
		return nil
	}

	patternKey := fmt.Sprintf("%s:first-filter", r.createPatternKey(valueNode, filepath))
	processedPatterns[patternKey] = true

	meta := r.Meta()
	return &Finding{
		RuleID:   meta.ID,
		Message:  "Using `(first (filter ...))` in let binding is inefficient. Consider using `(some ...)` which stops after the first match.",
		Filepath: filepath,
		Location: valueNode.Location,
		Severity: meta.Severity,
	}
}

func (r *InefficientFilteringRule) isFirstFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "first")
}

func (r *InefficientFilteringRule) isLastFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "last")
}

func (r *InefficientFilteringRule) isCountFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "count")
}

func (r *InefficientFilteringRule) isEmptyFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "empty?")
}

func (r *InefficientFilteringRule) isNotEmptyFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "not-empty")
}

func (r *InefficientFilteringRule) isWrapperFilter(node *reader.RichNode, wrapperFunc string) bool {
	if len(node.Children) != 2 {
		return false
	}

	outerFunc := node.Children[0]
	innerCall := node.Children[1]

	return outerFunc.Type == reader.NodeSymbol &&
		outerFunc.Value == wrapperFunc &&
		r.isFilterCall(innerCall)
}

func (r *InefficientFilteringRule) isMapFilter(node *reader.RichNode) bool {
	return r.isThreeArgPattern(node, "map", 2)
}

func (r *InefficientFilteringRule) isTakeFilter(node *reader.RichNode) bool {
	return r.isThreeArgPattern(node, "take", 2)
}

func (r *InefficientFilteringRule) isThreeArgPattern(node *reader.RichNode, funcName string, filterArgIndex int) bool {
	if len(node.Children) != 3 {
		return false
	}

	outerFunc := node.Children[0]
	filterArg := node.Children[filterArgIndex]

	return outerFunc.Type == reader.NodeSymbol &&
		outerFunc.Value == funcName &&
		r.isFilterCall(filterArg)
}

func (r *InefficientFilteringRule) isFilterCall(node *reader.RichNode) bool {
	return node.Type == reader.NodeList &&
		len(node.Children) >= 3 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "filter" || node.Children[0].Value == "remove")
}

func (r *InefficientFilteringRule) isThreadingMacro(node *reader.RichNode) bool {
	return len(node.Children) >= 3 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "->>" || node.Children[0].Value == "->")
}

func (r *InefficientFilteringRule) extractFilterOperations(node *reader.RichNode) []string {
	var operations []string

	for i := 2; i < len(node.Children); i++ {
		child := node.Children[i]

		if child.Type == reader.NodeList && len(child.Children) >= 1 {
			if funcNode := child.Children[0]; funcNode.Type == reader.NodeSymbol {
				if r.isFilterOperation(funcNode.Value) {
					operations = append(operations, funcNode.Value)
				}
			}
		} else if child.Type == reader.NodeSymbol && r.isFilterOperation(child.Value) {
			operations = append(operations, child.Value)
		}
	}

	return operations
}

func (r *InefficientFilteringRule) isFilterOperation(op string) bool {
	return op == "filter" || op == "remove"
}

func (r *InefficientFilteringRule) hasConsecutiveFilters(ops []string) bool {
	filterCount := 0
	for _, op := range ops {
		if op == "filter" {
			filterCount++
		}
	}
	return filterCount >= 2
}

func (r *InefficientFilteringRule) hasFilterRemovePattern(ops []string) bool {
	for i := 0; i < len(ops)-1; i++ {
		current := ops[i]
		next := ops[i+1]
		if (current == "filter" && next == "remove") || (current == "remove" && next == "filter") {
			return true
		}
	}
	return false
}

func (r *InefficientFilteringRule) isSecondFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "second")
}

func (r *InefficientFilteringRule) isNthFilter(node *reader.RichNode) bool {
	return r.isThreeArgPattern(node, "nth", 1)
}

func (r *InefficientFilteringRule) isSeqFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "seq")
}

func (r *InefficientFilteringRule) isNilSeqFilter(node *reader.RichNode) bool {
	if len(node.Children) != 2 {
		return false
	}

	outerFunc := node.Children[0]
	innerCall := node.Children[1]

	return outerFunc.Type == reader.NodeSymbol &&
		outerFunc.Value == "nil?" &&
		r.isSeqFilter(innerCall)
}

func (r *InefficientFilteringRule) isNotSeqFilter(node *reader.RichNode) bool {
	if len(node.Children) != 2 {
		return false
	}

	outerFunc := node.Children[0]
	innerCall := node.Children[1]

	return outerFunc.Type == reader.NodeSymbol &&
		outerFunc.Value == "not" &&
		r.isSeqFilter(innerCall)
}

func (r *InefficientFilteringRule) isDistinctFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "distinct")
}

func (r *InefficientFilteringRule) isSortFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "sort")
}

func (r *InefficientFilteringRule) isReverseFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "reverse")
}

func (r *InefficientFilteringRule) isDoallFilter(node *reader.RichNode) bool {
	return r.isWrapperFilter(node, "doall")
}

func init() {
	RegisterRule(&InefficientFilteringRule{})
}
