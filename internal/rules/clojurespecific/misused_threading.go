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
	direction, ok := resolvedThreadMacroDirection(node)
	if !ok || r.IsInside(context, "__non-evaluated__") {
		return nil
	}

	opposite := threadFirst
	if direction == threadFirst {
		opposite = threadLast
	}

	oppositeSteps := 0
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
		}
	}

	if oppositeSteps < 2 || matchingSteps != 0 {
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
