package clojurespecific

import (
	"github.com/thlaurentino/arit/internal/rules"
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type ImproperEmptinessCheckRule struct {
	rules.Rule
}

func (r *ImproperEmptinessCheckRule) Meta() rules.Rule {
	return rules.Rule{
		ID:          "improper-emptiness-check",
		Name:        "Improper Emptiness Check",
		Description: "Detects improper ways of checking for collection emptiness. Recommends using `(seq coll)` for non-emptiness and `(empty? coll)` for emptiness.",
		Severity:    rules.SeverityHint,
	}
}

func (r *ImproperEmptinessCheckRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if (node.Type != reader.NodeList && node.Type != reader.NodeFnLiteral) || len(node.Children) == 0 {
		return nil
	}

	if len(node.Children) >= 2 && node.Children[0].Type == reader.NodeSymbol {
		op := node.Children[0].Value

		if op == "not" || op == "when-not" || op == "if-not" {
			arg := node.Children[1]
			if arg.Type == reader.NodeList && len(arg.Children) == 2 &&
				arg.Children[0].Type == reader.NodeSymbol && arg.Children[0].Value == "empty?" {

				collectionExpr := arg.Children[1].Value
				opMap := map[string]string{
					"not":      fmt.Sprintf("(seq %s)", collectionExpr),
					"when-not": fmt.Sprintf("(when (seq %s) ...)", collectionExpr),
					"if-not":   fmt.Sprintf("(if (seq %s) ...)", collectionExpr),
				}

				return &rules.Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Improper emptiness check: `(%s (empty? %s))`. Consider using `%s`.", op, collectionExpr, opMap[op]),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}

		if op == "zero?" || op == "pos?" {
			arg := node.Children[1]
			if arg.Type == reader.NodeList && len(arg.Children) == 2 &&
				arg.Children[0].Type == reader.NodeSymbol && arg.Children[0].Value == "count" {
				
				collectionExpr := arg.Children[1].Value
				var replacement string
				if op == "zero?" {
					replacement = fmt.Sprintf("(empty? %s)", collectionExpr)
				} else {
					replacement = fmt.Sprintf("(seq %s)", collectionExpr)
				}
				
				return &rules.Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Improper emptiness check: `(%s (count %s))`. Consider using `%s`.", op, collectionExpr, replacement),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}

		if len(node.Children) == 3 && (op == "=" || op == "==" || op == "not=" || op == "<" || op == ">" || op == "<=" || op == ">=") {
			arg1 := node.Children[1]
			arg2 := node.Children[2]

			isCount := func(n *reader.RichNode) bool {
				return n.Type == reader.NodeList && len(n.Children) == 2 &&
					n.Children[0].Type == reader.NodeSymbol && n.Children[0].Value == "count"
			}
			isNum := func(n *reader.RichNode, v string) bool {
				return n.Type == reader.NodeNumber && n.Value == v
			}

			var collectionExpr string
			var pattern string 

			if isCount(arg1) {
				collectionExpr = arg1.Children[1].Value
				if isNum(arg2, "0") {
					switch op {
					case "=", "==":
						pattern = "count=0"
					case "not=":
						pattern = "count!=0"
					case ">":
						pattern = "count>0"
					}
				} else if isNum(arg2, "1") {
					if op == ">=" {
						pattern = "count>=1"
					}
				}
			} else if isCount(arg2) {
				collectionExpr = arg2.Children[1].Value
				if isNum(arg1, "0") {
					switch op {
					case "=", "==":
						pattern = "count=0"
					case "not=":
						pattern = "count!=0"
					case "<":
						pattern = "count>0"
					}
				} else if isNum(arg1, "1") {
					if op == "<=" {
						pattern = "count>=1"
					}
				}
			}

			if pattern != "" {
				var replacement string
				if pattern == "count=0" {
					replacement = fmt.Sprintf("(empty? %s)", collectionExpr)
				} else { 
					replacement = fmt.Sprintf("(seq %s)", collectionExpr)
				}

				return &rules.Finding{
					RuleID:   r.ID,
					Message:  fmt.Sprintf("Improper emptiness check: using `%s` with `count`. Consider using `%s`.", op, replacement),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	return nil
}

func init() {
	rules.RegisterRule(&ImproperEmptinessCheckRule{
		Rule: rules.Rule{
			ID:          "improper-emptiness-check",
			Name:        "Improper Emptiness Check",
			Description: "Detects improper ways of checking for collection emptiness. Recommends using `(seq coll)` for non-emptiness and `(empty? coll)` for emptiness.",
			Severity:    rules.SeverityHint,
		},
	})
}