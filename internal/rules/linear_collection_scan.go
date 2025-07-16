package rules

import (
	"fmt"
	"github.com/thlaurentino/arit/internal/reader"
)

// LinearCollectionScanRule detects inefficient linear scanning patterns in collections
type LinearCollectionScanRule struct {
	Rule
}

// PatternType represents different types of linear scan patterns
type PatternType int

const (
	PatternManualLoop PatternType = iota
	PatternChainedOperations
	PatternInefficient
	PatternRedundant
	PatternComposition
)

// LinearScanPattern represents a detected inefficient pattern
type LinearScanPattern struct {
	Type            PatternType
	Function        string
	Message         string
	Suggestion      string
	Severity        Severity
	ConfidenceLevel int // 1-100
}

// Common inefficient patterns and their better alternatives
var linearScanPatterns = map[string]LinearScanPattern{
	// Manual loops that can be replaced with built-ins
	"loop-find": {
		Type:            PatternManualLoop,
		Function:        "manual-find-loop",
		Message:         "Manual loop for finding elements can be replaced with 'some' or 'filter'",
		Suggestion:      "Use (some #(when (pred %) %) coll) or (filter pred coll)",
		Severity:        SeverityHint,
		ConfidenceLevel: 90,
	},
	"loop-count": {
		Type:            PatternManualLoop,
		Function:        "manual-count-loop",
		Message:         "Manual loop for counting can be replaced with 'count' or 'transduce'",
		Suggestion:      "Use (count (filter pred coll)) or (transduce (filter pred) + coll)",
		Severity:        SeverityHint,
		ConfidenceLevel: 95,
	},

	// Chained operations that create multiple passes
	"filter-first": {
		Type:            PatternChainedOperations,
		Function:        "filter-then-first",
		Message:         "Using 'first' after 'filter' creates unnecessary intermediate collection",
		Suggestion:      "Use (some #(when (pred %) %) coll) for early termination",
		Severity:        SeverityInfo,
		ConfidenceLevel: 85,
	},
	"count-filter": {
		Type:            PatternChainedOperations,
		Function:        "count-after-filter",
		Message:         "Counting filtered results can be done in single pass",
		Suggestion:      "Use (transduce (filter pred) (completing (fn [acc _] (inc acc))) 0 coll)",
		Severity:        SeverityInfo,
		ConfidenceLevel: 80,
	},
	"multiple-filters": {
		Type:            PatternChainedOperations,
		Function:        "multiple-filter-chains",
		Message:         "Multiple chained filters can be combined into single filter",
		Suggestion:      "Combine predicates: (filter (fn [x] (and (pred1 x) (pred2 x))) coll)",
		Severity:        SeverityInfo,
		ConfidenceLevel: 75,
	},

	// Inefficient operations
	"sort-for-min-max": {
		Type:            PatternInefficient,
		Function:        "sort-for-extremes",
		Message:         "Sorting collection just to find min/max is inefficient",
		Suggestion:      "Use (reduce min coll) or (reduce max coll)",
		Severity:        SeverityWarning,
		ConfidenceLevel: 95,
	},
	"membership-with-filter": {
		Type:            PatternInefficient,
		Function:        "filter-for-membership",
		Message:         "Using 'filter' for membership check is inefficient",
		Suggestion:      "Use (some #(= % target) coll) or convert to set first",
		Severity:        SeverityWarning,
		ConfidenceLevel: 90,
	},

	// Redundant operations
	"map-side-effects": {
		Type:            PatternRedundant,
		Function:        "map-for-side-effects",
		Message:         "Using 'map' for side effects is incorrect; use 'run!' or 'doseq'",
		Suggestion:      "Use (run! side-effect-fn coll) or (doseq [item coll] ...)",
		Severity:        SeverityWarning,
		ConfidenceLevel: 100,
	},
	"reduce-for-built-in": {
		Type:            PatternRedundant,
		Function:        "reduce-reimplementing-builtin",
		Message:         "Manual reduce implementation of built-in function",
		Suggestion:      "Use appropriate built-in function",
		Severity:        SeverityHint,
		ConfidenceLevel: 85,
	},
}

// NewLinearCollectionScanRule creates a new instance of the rule
func NewLinearCollectionScanRule() *LinearCollectionScanRule {
	return &LinearCollectionScanRule{
		Rule: Rule{
			ID:          "linear-collection-scan",
			Name:        "Linear Collection Scan",
			Description: "Detects inefficient linear scanning patterns in collections that can be optimized",
			Severity:    SeverityInfo,
		},
	}
}

func (r *LinearCollectionScanRule) Meta() Rule {
	return r.Rule
}

// Check analyzes a node for linear collection scan patterns
func (r *LinearCollectionScanRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	// Check different types of patterns
	if finding := r.checkManualLoops(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkChainedOperations(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkInefficient(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkRedundant(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkComposition(node, filepath); finding != nil {
		return finding
	}

	return nil
}

// checkManualLoops detects manual loop patterns that can be replaced with built-ins
func (r *LinearCollectionScanRule) checkManualLoops(node *reader.RichNode, filepath string) *Finding {
	if !r.isLoopConstruct(node) {
		return nil
	}

	// Detect manual find loops
	if r.isManualFindLoop(node) {
		return r.createFinding("loop-find", node, filepath)
	}

	// Detect manual count loops
	if r.isManualCountLoop(node) {
		return r.createFinding("loop-count", node, filepath)
	}

	return nil
}

// checkChainedOperations detects chained operations that create multiple passes
func (r *LinearCollectionScanRule) checkChainedOperations(node *reader.RichNode, filepath string) *Finding {
	funcName := r.getFunctionName(node)
	if funcName == "" {
		return nil
	}

	switch funcName {
	case "first":
		if r.isFilterFirst(node) {
			return r.createFinding("filter-first", node, filepath)
		}
	case "count":
		if r.isCountAfterFilter(node) {
			return r.createFinding("count-filter", node, filepath)
		}
	case "filter":
		if r.isMultipleFilters(node) {
			return r.createFinding("multiple-filters", node, filepath)
		}
	}

	return nil
}

// checkInefficient detects inefficient operations
func (r *LinearCollectionScanRule) checkInefficient(node *reader.RichNode, filepath string) *Finding {
	funcName := r.getFunctionName(node)
	if funcName == "" {
		return nil
	}

	switch funcName {
	case "first", "last":
		if r.isSortForMinMax(node) {
			return r.createFinding("sort-for-min-max", node, filepath)
		}
	case "filter":
		if r.isFilterForMembership(node) {
			return r.createFinding("membership-with-filter", node, filepath)
		}
	}

	return nil
}

// checkRedundant detects redundant operations
func (r *LinearCollectionScanRule) checkRedundant(node *reader.RichNode, filepath string) *Finding {
	funcName := r.getFunctionName(node)
	if funcName == "" {
		return nil
	}

	switch funcName {
	case "map":
		if r.isMapForSideEffects(node) {
			return r.createFinding("map-side-effects", node, filepath)
		}
	case "reduce":
		if r.isReduceForBuiltIn(node) {
			return r.createFinding("reduce-for-built-in", node, filepath)
		}
	}

	return nil
}

// checkComposition detects composition patterns that can be optimized
func (r *LinearCollectionScanRule) checkComposition(node *reader.RichNode, filepath string) *Finding {
	// Check for threading macro patterns
	if r.isThreadingMacro(node) {
		return r.checkThreadingPattern(node, filepath)
	}

	return nil
}

// Helper methods for pattern detection

func (r *LinearCollectionScanRule) isLoopConstruct(node *reader.RichNode) bool {
	if len(node.Children) == 0 {
		return false
	}

	funcName := r.getFunctionName(node)
	return funcName == "loop"
}

func (r *LinearCollectionScanRule) isManualFindLoop(node *reader.RichNode) bool {
	// Look for loop patterns that implement finding logic
	// This is a simplified check - in reality, you'd need more sophisticated AST analysis
	return r.containsPatternInBody(node, []string{"when", "seq", "first", "recur"})
}

func (r *LinearCollectionScanRule) isManualCountLoop(node *reader.RichNode) bool {
	// Look for loop patterns that implement counting logic
	return r.containsPatternInBody(node, []string{"inc", "recur"}) &&
		r.hasNumericAccumulator(node)
}

func (r *LinearCollectionScanRule) isFilterFirst(node *reader.RichNode) bool {
	// Check if this is (first (filter ...))
	if len(node.Children) != 2 {
		return false
	}

	arg := node.Children[1]
	return r.isCallToFunction(arg, "filter")
}

func (r *LinearCollectionScanRule) isCountAfterFilter(node *reader.RichNode) bool {
	// Check if this is (count (filter ...))
	if len(node.Children) != 2 {
		return false
	}

	arg := node.Children[1]
	return r.isCallToFunction(arg, "filter")
}

func (r *LinearCollectionScanRule) isMultipleFilters(node *reader.RichNode) bool {
	// Check if this filter is part of a chain of filters
	// This would require more context about the surrounding code
	return false // Simplified for now
}

func (r *LinearCollectionScanRule) isSortForMinMax(node *reader.RichNode) bool {
	// Check if this is (first (sort ...)) or (last (sort ...))
	if len(node.Children) != 2 {
		return false
	}

	arg := node.Children[1]
	return r.isCallToFunction(arg, "sort") || r.isCallToFunction(arg, "sort-by")
}

func (r *LinearCollectionScanRule) isFilterForMembership(node *reader.RichNode) bool {
	// Check if filter is used just for membership testing
	// This would require analyzing the predicate function
	return false // Simplified for now
}

func (r *LinearCollectionScanRule) isMapForSideEffects(node *reader.RichNode) bool {
	// Check if map is used for side effects (not transforming data)
	// This is complex to detect statically
	return false // Simplified for now
}

func (r *LinearCollectionScanRule) isReduceForBuiltIn(node *reader.RichNode) bool {
	// Check if reduce is implementing something like count, sum, etc.
	if len(node.Children) < 3 {
		return false
	}

	// Look for common patterns like (reduce + 0 coll) which could be (reduce + coll)
	return r.isSimpleAggregation(node)
}

func (r *LinearCollectionScanRule) isThreadingMacro(node *reader.RichNode) bool {
	funcName := r.getFunctionName(node)
	return funcName == "->>" || funcName == "->"
}

func (r *LinearCollectionScanRule) checkThreadingPattern(node *reader.RichNode, filepath string) *Finding {
	// Analyze threading patterns for optimization opportunities
	// This is a complex analysis that would look at the sequence of operations
	return nil // Simplified for now
}

// Utility methods

func (r *LinearCollectionScanRule) getFunctionName(node *reader.RichNode) string {
	if len(node.Children) == 0 {
		return ""
	}

	first := node.Children[0]
	if first.Type == reader.NodeSymbol {
		return first.Value
	}

	return ""
}

func (r *LinearCollectionScanRule) isCallToFunction(node *reader.RichNode, funcName string) bool {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}

	return r.getFunctionName(node) == funcName
}

func (r *LinearCollectionScanRule) containsPatternInBody(node *reader.RichNode, patterns []string) bool {
	// Recursively search for patterns in the node tree
	for _, child := range node.Children {
		if child.Type == reader.NodeSymbol {
			for _, pattern := range patterns {
				if child.Value == pattern {
					return true
				}
			}
		}
		if r.containsPatternInBody(child, patterns) {
			return true
		}
	}
	return false
}

func (r *LinearCollectionScanRule) hasNumericAccumulator(node *reader.RichNode) bool {
	// Check if the loop has a numeric accumulator (usually starts with 0)
	if len(node.Children) < 2 {
		return false
	}

	// Look for loop bindings
	bindings := node.Children[1]
	if bindings.Type != reader.NodeVector {
		return false
	}

	// Check if any binding is initialized with 0
	for i := 1; i < len(bindings.Children); i += 2 {
		if bindings.Children[i].Type == reader.NodeNumber &&
			bindings.Children[i].Value == "0" {
			return true
		}
	}

	return false
}

func (r *LinearCollectionScanRule) isSimpleAggregation(node *reader.RichNode) bool {
	// Check if this is a simple aggregation like (reduce + 0 coll)
	if len(node.Children) < 3 {
		return false
	}

	fn := node.Children[1]
	if fn.Type == reader.NodeSymbol {
		aggregationFns := []string{"+", "*", "min", "max", "and", "or"}
		for _, aggFn := range aggregationFns {
			if fn.Value == aggFn {
				return true
			}
		}
	}

	return false
}

func (r *LinearCollectionScanRule) createFinding(patternKey string, node *reader.RichNode, filepath string) *Finding {
	pattern, exists := linearScanPatterns[patternKey]
	if !exists {
		return nil
	}

	message := fmt.Sprintf("%s. %s", pattern.Message, pattern.Suggestion)

	return &Finding{
		RuleID:   r.ID,
		Message:  message,
		Filepath: filepath,
		Location: node.Location,
		Severity: pattern.Severity,
	}
}

// init registers the rule
func init() {
	RegisterRule(NewLinearCollectionScanRule())
}
