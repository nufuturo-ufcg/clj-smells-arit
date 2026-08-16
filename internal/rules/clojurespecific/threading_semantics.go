package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/reader"
)

type threadDirection string

const (
	threadFirst  threadDirection = "->"
	threadLast   threadDirection = "->>"
	threadEither threadDirection = "either"
)

type threadingSpec struct {
	direction threadDirection
	minArgs   int
}

// Only functions with a stable, documented primary data position are
// included. Canonical names prevent aliases and same-named project functions
// from being assigned clojure.core semantics.
var threadingSpecs = map[string]threadingSpec{
	"clojure.core/map":        {threadLast, 2},
	"clojure.core/mapv":       {threadLast, 2},
	"clojure.core/filter":     {threadLast, 2},
	"clojure.core/filterv":    {threadLast, 2},
	"clojure.core/remove":     {threadLast, 2},
	"clojure.core/keep":       {threadLast, 2},
	"clojure.core/mapcat":     {threadLast, 2},
	"clojure.core/reduce":     {threadLast, 2},
	"clojure.core/reduce-kv":  {threadLast, 3},
	"clojure.core/sort":       {threadLast, 1},
	"clojure.core/sort-by":    {threadLast, 2},
	"clojure.core/group-by":   {threadLast, 2},
	"clojure.core/partition":  {threadLast, 2},
	"clojure.core/take":       {threadLast, 2},
	"clojure.core/drop":       {threadLast, 2},
	"clojure.core/take-while": {threadLast, 2},
	"clojure.core/drop-while": {threadLast, 2},
	"clojure.core/concat":     {threadLast, 1},

	"clojure.core/assoc":       {threadFirst, 3},
	"clojure.core/dissoc":      {threadFirst, 2},
	"clojure.core/update":      {threadFirst, 3},
	"clojure.core/merge":       {threadFirst, 1},
	"clojure.core/select-keys": {threadFirst, 2},
	"clojure.core/get":         {threadFirst, 2},
	"clojure.core/get-in":      {threadFirst, 2},
	"clojure.core/assoc-in":    {threadFirst, 3},
	"clojure.core/update-in":   {threadFirst, 3},
	"clojure.core/conj":        {threadFirst, 2},

	"clojure.core/distinct": {threadEither, 1},
	"clojure.core/vec":      {threadEither, 1},
	"clojure.core/set":      {threadEither, 1},
	"clojure.core/seq":      {threadEither, 1},
	"clojure.core/keys":     {threadEither, 1},
	"clojure.core/vals":     {threadEither, 1},
	"clojure.core/first":    {threadEither, 1},
	"clojure.core/last":     {threadEither, 1},
	"clojure.core/rest":     {threadEither, 1},
	"clojure.core/next":     {threadEither, 1},
	"clojure.core/reverse":  {threadEither, 1},
	"clojure.core/count":    {threadEither, 1},
	"clojure.core/boolean":  {threadEither, 1},
	"clojure.core/name":     {threadEither, 1},
	"clojure.core/keyword":  {threadEither, 1},
	"clojure.core/identity": {threadEither, 1},
	"clojure.core/inc":      {threadEither, 1},
	"clojure.core/dec":      {threadEither, 1},
	"clojure.core/str":      {threadEither, 1},
	"clojure.core/flatten":  {threadLast, 1},
	"clojure.core/into":     {threadEither, 2},

	"clojure.string/trim":       {threadEither, 1},
	"clojure.string/upper-case": {threadEither, 1},
	"clojure.string/lower-case": {threadEither, 1},
	"clojure.string/capitalize": {threadEither, 1},
	"clojure.string/replace":    {threadFirst, 3},
	"clojure.string/split":      {threadFirst, 2},
	"clojure.string/join":       {threadLast, 1},
}

func resolvedThreadingSpec(node *reader.RichNode) (threadingSpec, bool) {
	if !isCall(node) {
		return threadingSpec{}, false
	}
	return resolvedThreadingHeadSpec(node.Children[0])
}

func unwrapStepHead(step *reader.RichNode) *reader.RichNode {
	if step == nil {
		return nil
	}
	if step.Type == reader.NodeFnLiteral && len(step.Children) > 0 {
		return step.Children[0]
	}
	if step.Type == reader.NodeList && len(step.Children) == 1 && step.Children[0].Type == reader.NodeFnLiteral && len(step.Children[0].Children) > 0 {
		return step.Children[0].Children[0]
	}
	if step.Type == reader.NodeList && len(step.Children) > 0 {
		return step.Children[0]
	}
	return nil
}

func resolvedThreadingStepSpec(node *reader.RichNode) (threadingSpec, bool) {
	head := unwrapStepHead(node)
	if head == nil {
		return threadingSpec{}, false
	}
	return resolvedThreadingHeadSpec(head)
}

func resolvedThreadingHeadSpec(head *reader.RichNode) (threadingSpec, bool) {
	if head == nil || head.Type != reader.NodeSymbol {
		return threadingSpec{}, false
	}
	if head.Resolution != nil {
		if spec, ok := threadingSpecs[head.Resolution.CanonicalName]; ok &&
			head.Resolution.Kind != reader.ResolutionLocal && head.Resolution.Kind != reader.ResolutionUnresolved {
			return spec, true
		}
		if head.Resolution.Kind == reader.ResolutionLocal {
			return threadingSpec{}, false
		}
	}

	if spec, ok := threadingSpecs[head.Value]; ok {
		return spec, true
	}
	if spec, ok := threadingSpecs["clojure.core/"+head.Value]; ok {
		return spec, ok
	}
	return threadingSpec{}, false
}

func resolvedThreadMacroDirection(node *reader.RichNode) (threadDirection, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) < 3 ||
		node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
		return "", false
	}
	head := node.Children[0]
	if head.Resolution != nil && head.Resolution.Kind == reader.ResolutionLocal {
		return "", false
	}
	name := head.Value
	if head.Resolution != nil && head.Resolution.Kind != reader.ResolutionUnresolved {
		name = head.Resolution.CanonicalName
	}
	switch name {
	case "->", "some->", "clojure.core/->", "clojure.core/some->":
		return threadFirst, true
	case "->>", "some->>", "clojure.core/->>", "clojure.core/some->>":
		return threadLast, true
	default:
		return "", false
	}
}

func isCall(node *reader.RichNode) bool {
	return node != nil && node.Type == reader.NodeList && len(node.Children) > 1 &&
		node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol
}

func callArgCount(node *reader.RichNode) int { return len(node.Children) - 1 }

func dataArgumentIndex(spec threadingSpec, node *reader.RichNode) (int, bool) {
	switch spec.direction {
	case threadFirst:
		return 1, len(node.Children) > 1
	case threadLast:
		return len(node.Children) - 1, len(node.Children) > 1
	case threadEither:
		if callArgCount(node) == 1 {
			return 1, true
		}
		return len(node.Children) - 1, len(node.Children) > 1
	default:
		return 0, false
	}
}

func mergeDirection(current *threadDirection, next threadDirection) bool {
	if next == threadEither {
		return true
	}
	if *current == threadEither {
		*current = next
		return true
	}
	return *current == next
}
