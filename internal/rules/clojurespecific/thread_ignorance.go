package clojurespecific

import (
	"fmt"
	"regexp"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ThreadIgnoranceRule struct {
	rules.Rule
	MinNestingDepth int `json:"min_nesting_depth" yaml:"min_nesting_depth"`
	MinLetChain     int `json:"min_let_chain" yaml:"min_let_chain"`
}

func (r *ThreadIgnoranceRule) Meta() rules.Rule { return r.Rule }

type pipeline struct {
	depth     int
	direction threadDirection
}

func (r *ThreadIgnoranceRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || r.excludedContext(context) || isNestedPipelineContinuation(node, context) {
		return nil
	}

	if p, ok := nestedPipeline(node); ok && p.depth >= r.nestingThreshold() {
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"Safe %s pipeline detected across %d resolved calls. Each call has a single nested data argument in the required threading position.",
				p.direction, p.depth,
			),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	if p, ok := r.letPipeline(node); ok {
		return &rules.Finding{
			RuleID: r.ID,
			Message: fmt.Sprintf(
				"Safe %s pipeline detected in %d let bindings. Intermediates are generic, used once, and the final binding is returned directly.",
				p.direction, p.depth,
			),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *ThreadIgnoranceRule) excludedContext(context map[string]interface{}) bool {
	if r.IsInside(context, "__non-evaluated__", "->", "->>", "some->", "some->>", "as->", "cond->", "cond->>", "fn", "delay", "lazy-seq") {
		return true
	}
	parent, _ := context["parent"].(*reader.RichNode)
	return parent != nil && parent.Type == reader.NodeVector
}

func isNestedPipelineContinuation(node *reader.RichNode, context map[string]interface{}) bool {
	parent, _ := context["parent"].(*reader.RichNode)
	if parent == nil {
		return false
	}
	if _, ok := resolvedThreadingSpec(parent); !ok {
		return false
	}
	for _, index := range directListArguments(parent) {
		if parent.Children[index] == node {
			return true
		}
	}
	return false
}

func (r *ThreadIgnoranceRule) nestingThreshold() int {
	if r.MinNestingDepth < 3 {
		return 3
	}
	return r.MinNestingDepth
}

func (r *ThreadIgnoranceRule) letThreshold() int {
	// MinLetChain historically counted links, so two means three forms.
	if r.MinLetChain < 2 {
		return 3
	}
	return r.MinLetChain + 1
}

func nestedPipeline(node *reader.RichNode) (pipeline, bool) {
	if !isCall(node) {
		return pipeline{}, false
	}
	current := node
	depth := 0
	direction := threadEither
	for isCall(current) {
		spec, ok := resolvedThreadingSpec(current)
		if !ok || callArgCount(current) < spec.minArgs {
			break
		}
		if spec.direction != threadEither && !mergeDirection(&direction, spec.direction) {
			break
		}
		dataIndex, ok := dataArgumentIndex(spec, current)
		if !ok {
			break
		}
		depth++
		current = current.Children[dataIndex]
	}
	if depth < 3 {
		return pipeline{}, false
	}
	if direction == threadEither {
		direction = threadFirst
	}
	for _, child := range node.Children[1:] {
		if _, ok := resolvedThreadingSpec(child); !ok {
			continue
		}
	}
	return pipeline{depth: depth, direction: direction}, true
}

func (r *ThreadIgnoranceRule) letPipeline(node *reader.RichNode) (pipeline, bool) {
	if node == nil || node.Type != reader.NodeList || len(node.Children) != 3 ||
		!resolvesToCoreForm(node, "let") {
		return pipeline{}, false
	}
	bindings := node.Children[1]
	if bindings == nil || bindings.Type != reader.NodeVector || len(bindings.Children)%2 != 0 {
		return pipeline{}, false
	}
	steps := len(bindings.Children) / 2
	if steps < r.letThreshold() {
		return pipeline{}, false
	}

	names := make([]string, 0, steps)
	direction := threadEither
	for i := 0; i < len(bindings.Children); i += 2 {
		nameNode, expr := bindings.Children[i], bindings.Children[i+1]
		if nameNode == nil || nameNode.Type != reader.NodeSymbol || nameNode.TypeHint != "" || expr == nil {
			return pipeline{}, false
		}
		spec, ok := resolvedThreadingSpec(expr)
		if !ok || callArgCount(expr) < spec.minArgs {
			return pipeline{}, false
		}

		dataIndex, ok := dataArgumentIndex(spec, expr)
		if !ok || (len(expr.Children) > dataIndex && expr.Children[dataIndex].Type == reader.NodeList) {
			return pipeline{}, false
		}
		if i > 0 {
			previous := names[len(names)-1]
			if !isExactSymbol(expr.Children[dataIndex], previous) || countSymbolOccurrences(expr, previous) != 1 {
				return pipeline{}, false
			}
			if spec.direction != threadEither && !mergeDirection(&direction, spec.direction) {
				return pipeline{}, false
			}
		}
		names = append(names, nameNode.Value)
	}

	body := node.Children[2]
	lastName := names[len(names)-1]
	if !isExactSymbol(body, lastName) {
		return pipeline{}, false
	}
	for _, name := range names {
		if countSymbolOccurrences(node, name) != 2 {
			return pipeline{}, false
		}
	}
	if direction == threadEither {
		direction = threadFirst
	}
	return pipeline{depth: steps, direction: direction}, true
}

func directListArguments(node *reader.RichNode) []int {
	indices := []int{}
	for i := 1; i < len(node.Children); i++ {
		if node.Children[i] != nil && node.Children[i].Type == reader.NodeList {
			indices = append(indices, i)
		}
	}
	return indices
}

func specAcceptsIndex(spec threadingSpec, node *reader.RichNode, index int) bool {
	switch spec.direction {
	case threadFirst:
		return index == 1
	case threadLast:
		return index == len(node.Children)-1
	case threadEither:
		return callArgCount(node) == 1 && index == 1
	default:
		return false
	}
}

func isExactSymbol(node *reader.RichNode, name string) bool {
	return node != nil && node.Type == reader.NodeSymbol && node.Value == name
}

func countSymbolOccurrences(node *reader.RichNode, symbol string) int {
	if node == nil {
		return 0
	}
	count := 0
	if isExactSymbol(node, symbol) {
		count++
	}
	for _, child := range node.Children {
		count += countSymbolOccurrences(child, symbol)
	}
	return count
}

var genericIntermediateName = regexp.MustCompile(`^(?:step[0-9]+|tmp[0-9]*|temp[0-9]*|intermediate[0-9]*|value[0-9]+|v[0-9]+)$`)

func isGenericIntermediate(name string) bool {
	return genericIntermediateName.MatchString(name)
}

func init() {
	rules.RegisterRule(&ThreadIgnoranceRule{
		Rule: rules.Rule{
			ID:          "thread-ignorance",
			Name:        "Thread Ignorance",
			Description: "Detects linear pipelines only when resolved call positions prove a semantics-preserving -> or ->> rewrite.",
			Severity:    rules.SeverityHint,
		},
		MinNestingDepth: 3,
		MinLetChain:     2,
	})
}
