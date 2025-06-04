package rules

import (
	"fmt"

	"github.com/thlaurentino/arit/internal/reader"
)

type StringMapKeysRule struct {
	Rule
}

func (r *StringMapKeysRule) Meta() Rule {
	return r.Rule
}

func init() {
	RegisterRule(&StringMapKeysRule{
		Rule: Rule{
			ID:          "string-map-keys",
			Name:        "String Keys in Map Literal",
			Description: "Map literals should use keywords (:key) instead of strings (\"key\") as keys for better performance and idiomatic style.",

			Severity: SeverityInfo,
		},
	})
}

func (r *StringMapKeysRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeMap {
		return nil
	}

	if len(node.Children)%2 != 0 {

		return nil
	}

	for i := 0; i < len(node.Children); i += 2 {
		keyNode := node.Children[i]

		if keyNode.InferredType == "String" {
			return &Finding{
				RuleID:   r.ID,
				Message:  fmt.Sprintf("Map literal uses string key %q instead of a keyword. Consider using ':%s' for idiomatic Clojure.", keyNode.Value, keyNode.Value),
				Filepath: filepath,
				Location: keyNode.Location,
				Severity: r.Severity,
			}

		}
	}

	return nil
}
