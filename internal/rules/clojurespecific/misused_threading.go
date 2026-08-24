package clojurespecific

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type MisusedThreadingRule struct {
	rules.Rule
}

func (r *MisusedThreadingRule) Meta() rules.Rule { return r.Rule }

// Check reports only a consistent, resolved positional contradiction. A lone
// step is not enough evidence: Clojure functions can intentionally receive the
// threaded value in a role other than their conventional data argument.
func (r *MisusedThreadingRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if rules.IsPathAllowed(context, r.Meta().ID, filepath) {
		return nil
	}
	direction, ok := resolvedThreadMacroDirection(node)
	if !ok || r.IsInside(context, "__non-evaluated__") {
		return nil
	}

	opposite := threadFirst
	if direction == threadFirst {
		opposite = threadLast
	}

	oppositeSteps := 0
	provableOppositeSteps := 0
	matchingSteps := 0
	for _, step := range node.Children[2:] {
		head := unwrapStepHead(step)
		spec, resolved := resolvedThreadingStepSpec(step)
		if !resolved || spec.direction == threadEither {
			continue
		}

		if step.Type == reader.NodeList && len(step.Children) > 0 && step.Children[0] == head && len(step.Children) < spec.minArgs {
			continue
		}

		if spec.direction == direction {
			matchingSteps++
		} else if spec.direction == opposite {
			oppositeSteps++
			if provableThreadingContradiction(step, spec, direction) {
				provableOppositeSteps++
			}
		}
	}

	if oppositeSteps < 2 || provableOppositeSteps < 2 || matchingSteps != 0 {
		return nil
	}

	return &rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Threading macro `%s` inserts the value in the %s argument, but %d resolved pipeline steps consistently use functions whose primary data argument is the %s. Use explicit positioning or review whether `%s` expresses this pipeline more accurately.",
			direction, threadPosition(direction), oppositeSteps, threadPosition(opposite), opposite,
		),
		Filepath: filepath,
		Location: node.Location,
		Severity: r.Severity,
	}
}

func provableThreadingContradiction(step *reader.RichNode, spec threadingSpec, direction threadDirection) bool {
	if step == nil || step.Type != reader.NodeList || len(step.Children) < 2 {
		return false
	}
	if direction == threadFirst && spec.direction == threadLast {
		// -> places the pipeline value before the explicit arguments. A known
		// function in the first explicit position makes the contradiction
		// concrete for collection functions such as map/filter.
		return isFunctionLike(step.Children[1])
	}
	if direction == threadLast && spec.direction == threadFirst {
		first := step.Children[1]
		if first == nil {
			return false
		}
		if specName := canonicalStepName(step); specName == "clojure.core/select-keys" && first.Type == reader.NodeVector {
			return true
		}
		switch first.Type {
		case reader.NodeKeyword, reader.NodeString, reader.NodeNumber, reader.NodeBool, reader.NodeNil:
			return true
		}
	}
	return false
}

func isFunctionLike(node *reader.RichNode) bool {
	if node == nil {
		return false
	}
	if node.Type == reader.NodeFnLiteral {
		return true
	}
	if node.Type != reader.NodeSymbol {
		return false
	}
	name := node.Value
	if node.Resolution != nil {
		name = node.Resolution.CanonicalName
	}
	for _, candidate := range []string{"inc", "dec", "identity", "even?", "odd?", "neg?", "pos?", "some?", "nil?", "string?", "number?", "true?", "false?"} {
		if name == candidate || name == "clojure.core/"+candidate {
			return true
		}
	}
	return false
}

func canonicalStepName(step *reader.RichNode) string {
	if step == nil || len(step.Children) == 0 || step.Children[0] == nil || step.Children[0].Type != reader.NodeSymbol {
		return ""
	}
	if step.Children[0].Resolution != nil {
		return step.Children[0].Resolution.CanonicalName
	}
	return step.Children[0].Value
}

func threadPosition(direction threadDirection) string {
	if direction == threadFirst {
		return "first"
	}
	return "last"
}

func init() {
	rules.RegisterRule(&MisusedThreadingRule{
		Rule: rules.Rule{
			ID:          "misused-threading",
			Name:        "Misused Threading",
			Description: "Detects threading pipelines only when resolved call semantics consistently contradict the macro's argument position.",
			Severity:    rules.SeverityWarning,
		},
	})
}
