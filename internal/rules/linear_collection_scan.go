package rules

import (
	"github.com/thlaurentino/arit/internal/reader"
)

type LinearCollectionScanRule struct {
	Rule
	processedLines map[string]map[int]bool
	lastFilepath   string
}

func NewLinearCollectionScanRule() *LinearCollectionScanRule {
	return &LinearCollectionScanRule{
		Rule: Rule{
			ID:          "linear-collection-scan",
			Name:        "Linear Collection Scan",
			Description: "Detects inefficient linear scanning patterns in collections that can be optimized",
			Severity:    SeverityInfo,
		},
		processedLines: make(map[string]map[int]bool),
	}
}

func (r *LinearCollectionScanRule) Meta() Rule {
	return r.Rule
}

func (r *LinearCollectionScanRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	currentLine := 0
	if node.Location != nil {
		currentLine = node.Location.StartLine
	}

	if currentLine <= 0 {
		return nil
	}

	if r.lastFilepath != filepath {
		r.processedLines = make(map[string]map[int]bool)
		r.lastFilepath = filepath
	}

	if r.processedLines[filepath] == nil {
		r.processedLines[filepath] = make(map[int]bool)
	}

	if r.processedLines[filepath][currentLine] {
		return nil
	}

	var finding *Finding

	if r.isNestedMapOrFilter(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Multiple nested map/filter detected. Prefer function composition or threading macros to avoid multiple collection passes.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityWarning,
		}
	}

	if finding == nil && r.isSortForMinMax(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using sort for min/max detected. Prefer 'reduce' or 'apply min/max' for efficiency.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityWarning,
		}
	}

	if finding == nil && r.isMapForSideEffectsPotential(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using map for side effects detected. Prefer 'run!' or 'doseq' for side effects.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityWarning,
		}
	}

	if finding == nil && r.isFilterForMembership(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using filter for membership detected. Prefer 'some' or 'set' for existence check.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityWarning,
		}
	}

	if finding == nil && r.isFirstOrLastAfterFilter(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using first/last after filter detected. Prefer 'some' for search or 'not-any?' for existence check.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityInfo,
		}
	}

	if finding == nil && r.isCountFilterExistence(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using count/filter for existence check detected. Prefer 'some' or 'not-any?' for efficiency.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityInfo,
		}
	}

	if finding == nil && r.isChainedFilters(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Chained filters detected. Combine predicates into a single filter.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityInfo,
		}
	}

	if finding == nil && r.isDeepNestingThreadingCandidate(node, 1) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Deep function nesting detected. Prefer threading macros (->, ->>) for clarity and efficiency.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityInfo,
		}
	}

	if finding == nil && r.isNestedConcat(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Nested concatenations detected. Prefer 'apply concat' or 'into' for efficiency.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityInfo,
		}
	}

	if finding == nil && r.isReduceReimplementingBuiltin(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using reduce to reimplement built-in function detected. Prefer the built-in function directly.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	if finding == nil && r.isTrivialFor(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Trivial for usage detected. Prefer 'map' or 'filter' for simple transformations.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	if finding == nil && r.isTakeDropSequence(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using take/drop in sequence detected. Prefer 'subvec' (for vectors) or dedicated function for efficiency.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	if finding == nil && r.isTakeRepeatedly(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using take with repeatedly detected. Prefer 'map' with 'range' for finite sequences.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	if finding == nil && r.isMapOnHashMap(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using map over hash-map detected. Order not guaranteed, prefer vector of pairs or mapv if order matters.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	if finding == nil && r.isCountForEmptiness(node) {
		finding = &Finding{
			RuleID:   r.ID,
			Message:  "Using count for emptiness check detected. Prefer 'empty?' or 'seq' for clarity and efficiency.",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	if finding == nil {
		finding = r.checkManualLoops(node, filepath)
	}
	if finding == nil {
		finding = r.checkChainedOperations(node, filepath)
	}
	if finding == nil {
		finding = r.checkInefficient(node, filepath)
	}
	if finding == nil {
		finding = r.checkRedundant(node, filepath)
	}

	if finding != nil {
		r.processedLines[filepath][currentLine] = true
		return finding
	}

	return nil
}

func (r *LinearCollectionScanRule) checkManualLoops(node *reader.RichNode, filepath string) *Finding {
	if !r.isLoopConstruct(node) {
		return nil
	}

	if r.isManualFindLoop(node) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Manual loop for finding elements can be replaced with 'some' or 'filter'. Use (some #(when (pred %) %) coll) or (filter pred coll)",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	if r.isManualCountLoop(node) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Manual loop for counting can be replaced with 'count' or 'transduce'. Use (count (filter pred coll)) or (transduce (filter pred) + coll)",
			Filepath: filepath,
			Location: node.Location,
			Severity: SeverityHint,
		}
	}

	return nil
}

func (r *LinearCollectionScanRule) checkChainedOperations(node *reader.RichNode, filepath string) *Finding {
	funcName := r.getFunctionName(node)
	if funcName == "" {
		return nil
	}

	switch funcName {
	case "first":
		if r.isFilterFirst(node) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Using 'first' after 'filter' creates unnecessary intermediate collection. Use (some #(when (pred %) %) coll) for early termination",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityInfo,
			}
		}
	case "count":
		if r.isCountAfterFilter(node) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Counting filtered results can be done in single pass. Use (transduce (filter pred) (completing (fn [acc _] (inc acc))) 0 coll)",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityInfo,
			}
		}
	case "filter":
		if r.isMultipleFilters(node) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Multiple chained filters can be combined into single filter. Combine predicates: (filter (fn [x] (and (pred1 x) (pred2 x))) coll)",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityInfo,
			}
		}
	}

	return nil
}

func (r *LinearCollectionScanRule) checkInefficient(node *reader.RichNode, filepath string) *Finding {
	funcName := r.getFunctionName(node)
	if funcName == "" {
		return nil
	}

	switch funcName {
	case "first", "last":
		if r.isSortForMinMax(node) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Sorting collection just to find min/max is inefficient. Use (reduce min coll) or (reduce max coll)",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityWarning,
			}
		}
	case "filter":
		if r.isFilterForMembership(node) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Using 'filter' for membership check is inefficient. Use (some #(= % target) coll) or convert to set first",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityWarning,
			}
		}
	}

	return nil
}

func (r *LinearCollectionScanRule) checkRedundant(node *reader.RichNode, filepath string) *Finding {
	funcName := r.getFunctionName(node)
	if funcName == "" {
		return nil
	}

	switch funcName {
	case "map":
		if r.isMapForSideEffects(node) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Using 'map' for side effects is incorrect; use 'run!' or 'doseq'. Use (run! side-effect-fn coll) or (doseq [item coll] ...)",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityWarning,
			}
		}
	case "reduce":
		if r.isReduceForBuiltIn(node) {
			return &Finding{
				RuleID:   r.ID,
				Message:  "Manual reduce implementation of built-in function. Use appropriate built-in function",
				Filepath: filepath,
				Location: node.Location,
				Severity: SeverityHint,
			}
		}
	}

	return nil
}

func (r *LinearCollectionScanRule) checkComposition(node *reader.RichNode, filepath string) *Finding {

	if r.isThreadingMacro(node) {
		return r.checkThreadingPattern(node, filepath)
	}

	return nil
}

func (r *LinearCollectionScanRule) isLoopConstruct(node *reader.RichNode) bool {
	if len(node.Children) == 0 {
		return false
	}

	funcName := r.getFunctionName(node)
	return funcName == "loop"
}

func (r *LinearCollectionScanRule) isManualFindLoop(node *reader.RichNode) bool {

	return r.containsPatternInBody(node, []string{"when", "seq", "first", "recur"})
}

func (r *LinearCollectionScanRule) isManualCountLoop(node *reader.RichNode) bool {

	return r.containsPatternInBody(node, []string{"inc", "recur"}) &&
		r.hasNumericAccumulator(node)
}

func (r *LinearCollectionScanRule) isFilterFirst(node *reader.RichNode) bool {

	if len(node.Children) != 2 {
		return false
	}

	arg := node.Children[1]
	return r.isCallToFunction(arg, "filter")
}

func (r *LinearCollectionScanRule) isCountAfterFilter(node *reader.RichNode) bool {

	if len(node.Children) != 2 {
		return false
	}

	arg := node.Children[1]
	return r.isCallToFunction(arg, "filter")
}

func (r *LinearCollectionScanRule) isMultipleFilters(node *reader.RichNode) bool {

	return false
}

func (r *LinearCollectionScanRule) isSortForMinMax(node *reader.RichNode) bool {

	if len(node.Children) != 2 {
		return false
	}

	arg := node.Children[1]
	return r.isCallToFunction(arg, "sort") || r.isCallToFunction(arg, "sort-by")
}

func (r *LinearCollectionScanRule) isMapForSideEffects(node *reader.RichNode) bool {

	return false
}

func (r *LinearCollectionScanRule) isReduceForBuiltIn(node *reader.RichNode) bool {

	if len(node.Children) < 3 {
		return false
	}

	return r.isSimpleAggregation(node)
}

func (r *LinearCollectionScanRule) isThreadingMacro(node *reader.RichNode) bool {
	funcName := r.getFunctionName(node)
	return funcName == "->>" || funcName == "->"
}

func (r *LinearCollectionScanRule) checkThreadingPattern(node *reader.RichNode, filepath string) *Finding {

	return nil
}

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

	if len(node.Children) < 2 {
		return false
	}

	bindings := node.Children[1]
	if bindings.Type != reader.NodeVector {
		return false
	}

	for i := 1; i < len(bindings.Children); i += 2 {
		if bindings.Children[i].Type == reader.NodeNumber &&
			bindings.Children[i].Value == "0" {
			return true
		}
	}

	return false
}

func (r *LinearCollectionScanRule) isSimpleAggregation(node *reader.RichNode) bool {

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

func init() {
	RegisterRule(NewLinearCollectionScanRule())
}

func (r *LinearCollectionScanRule) isNestedMapOrFilter(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "map" && funcName != "filter" && funcName != "remove" {
		return false
	}
	arg := node.Children[len(node.Children)-1]

	if arg.Type == reader.NodeList {
		innerFunc := r.getFunctionName(arg)
		if innerFunc == "map" || innerFunc == "filter" || innerFunc == "remove" {
			return true
		}
	}
	return false
}

func (r *LinearCollectionScanRule) isFirstOrLastAfterFilter(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 2 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "first" && funcName != "last" {
		return false
	}
	arg := node.Children[1]
	return r.isCallToFunction(arg, "filter")
}

func (r *LinearCollectionScanRule) isCountFilterExistence(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName == ">" || funcName == "pos?" {
		for _, arg := range node.Children[1:] {
			if arg.Type == reader.NodeList && r.isCountAfterFilter(arg) {
				return true
			}
		}
	}
	return false
}

func (r *LinearCollectionScanRule) isReduceReimplementingBuiltin(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "reduce" {
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

func (r *LinearCollectionScanRule) isMapForSideEffectsPotential(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "map" {
		return false
	}
	fn := node.Children[1]
	if fn.Type == reader.NodeSymbol && (fn.Value == "println" || fn.Value == "prn" || fn.Value == "print") {
		return true
	}
	return false
}

func (r *LinearCollectionScanRule) isFilterForMembership(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName == "not" && len(node.Children) == 2 {
		inner := node.Children[1]
		if inner.Type == reader.NodeList && r.getFunctionName(inner) == "empty?" && len(inner.Children) == 2 {
			return r.isCallToFunction(inner.Children[1], "filter")
		}
	}
	if funcName == "not-empty" && len(node.Children) == 2 {
		return r.isCallToFunction(node.Children[1], "filter")
	}
	return false
}

func (r *LinearCollectionScanRule) isChainedFilters(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "filter" {
		return false
	}
	arg := node.Children[len(node.Children)-1]
	return r.isCallToFunction(arg, "filter")
}

func (r *LinearCollectionScanRule) isDeepNestingThreadingCandidate(node *reader.RichNode, depth int) bool {
	if node == nil || node.Type != reader.NodeList || depth > 3 {
		return false
	}
	funcName := r.getFunctionName(node)
	candidates := map[string]bool{"map": true, "filter": true, "remove": true, "reduce": true, "mapcat": true}
	if !candidates[funcName] {
		return false
	}
	for _, child := range node.Children[1:] {
		if child.Type == reader.NodeList && r.isDeepNestingThreadingCandidate(child, depth+1) {
			return true
		}
	}
	return depth >= 3
}

func (r *LinearCollectionScanRule) isTrivialFor(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "for" {
		return false
	}
	bindings := node.Children[1]
	body := node.Children[2]
	if bindings.Type == reader.NodeVector && len(bindings.Children) == 2 && body.Type == reader.NodeList && len(body.Children) == 2 {
		return true
	}
	return false
}

func (r *LinearCollectionScanRule) isNestedConcat(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "concat" {
		return false
	}
	for _, arg := range node.Children[1:] {
		if arg.Type == reader.NodeList && r.getFunctionName(arg) == "concat" {
			return true
		}
	}
	return false
}

func (r *LinearCollectionScanRule) isTakeDropSequence(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 3 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "take" {
		return false
	}
	dropArg := node.Children[2]
	return r.isCallToFunction(dropArg, "drop")
}

func (r *LinearCollectionScanRule) isTakeRepeatedly(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 3 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "take" {
		return false
	}
	repArg := node.Children[2]
	return r.isCallToFunction(repArg, "repeatedly")
}

func (r *LinearCollectionScanRule) isMapOnHashMap(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName != "map" {
		return false
	}
	lastArg := node.Children[len(node.Children)-1]
	return lastArg.Type == reader.NodeMap
}

func (r *LinearCollectionScanRule) isCountForEmptiness(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}
	funcName := r.getFunctionName(node)
	if funcName == "=" || funcName == ">" {
		for _, arg := range node.Children[1:] {
			if arg.Type == reader.NodeList && r.getFunctionName(arg) == "count" {
				return true
			}
		}
	}
	return false
}
