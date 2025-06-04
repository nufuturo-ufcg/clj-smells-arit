package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type AccessingNonexistentMapFieldsRule struct {
	Rule
	CheckDirectKeywordAccess bool `json:"check_direct_keyword_access" yaml:"check_direct_keyword_access"`
	CheckThreadingMacros     bool `json:"check_threading_macros" yaml:"check_threading_macros"`
	CheckNestedAccess        bool `json:"check_nested_access" yaml:"check_nested_access"`
	MinNestingLevel          int  `json:"min_nesting_level" yaml:"min_nesting_level"`
}

func (r *AccessingNonexistentMapFieldsRule) Meta() Rule {
	return r.Rule
}

func (r *AccessingNonexistentMapFieldsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if finding := r.checkDirectKeywordAccess(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkThreadingMacroAccess(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkGetInAccess(node, filepath); finding != nil {
		return finding
	}

	if finding := r.checkNestedMapAccess(node, filepath); finding != nil {
		return finding
	}

	return nil
}

func (r *AccessingNonexistentMapFieldsRule) checkDirectKeywordAccess(node *reader.RichNode, filepath string) *Finding {
	if !r.CheckDirectKeywordAccess {
		return nil
	}

	if node.Type == reader.NodeList && len(node.Children) == 2 {
		firstChild := node.Children[0]
		secondChild := node.Children[1]

		if firstChild.Type == reader.NodeKeyword && secondChild.Type == reader.NodeSymbol {
			keyword := firstChild.Value
			mapVar := secondChild.Value

			if r.isCommonSafePattern(keyword, mapVar) {
				return nil
			}

			return &Finding{
				RuleID:   r.ID,
				Message:  r.formatDirectAccessMessage(keyword, mapVar),
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

func (r *AccessingNonexistentMapFieldsRule) checkThreadingMacroAccess(node *reader.RichNode, filepath string) *Finding {
	if !r.CheckThreadingMacros {
		return nil
	}

	if node.Type == reader.NodeList && len(node.Children) >= 3 {
		firstChild := node.Children[0]

		if firstChild.Type == reader.NodeSymbol && (firstChild.Value == "->" || firstChild.Value == "->>") {

			keywordCount := 0
			var keywords []string

			for i := 2; i < len(node.Children); i++ {
				child := node.Children[i]
				if child.Type == reader.NodeKeyword {
					keywordCount++
					keywords = append(keywords, child.Value)
				} else {
					break
				}
			}

			if keywordCount >= r.MinNestingLevel {
				return &Finding{
					RuleID:   r.ID,
					Message:  r.formatThreadingMessage(firstChild.Value, keywords),
					Filepath: filepath,
					Location: node.Location,
					Severity: r.Severity,
				}
			}
		}
	}

	return nil
}

func (r *AccessingNonexistentMapFieldsRule) checkGetInAccess(node *reader.RichNode, filepath string) *Finding {

	if node.Type == reader.NodeList && len(node.Children) == 3 {
		firstChild := node.Children[0]

		if firstChild.Type == reader.NodeSymbol && firstChild.Value == "get-in" {

			return &Finding{
				RuleID:   r.ID,
				Message:  "get-in without default value detected. Consider providing a default to handle missing keys safely.",
				Filepath: filepath,
				Location: node.Location,
				Severity: r.Severity,
			}
		}
	}

	return nil
}

func (r *AccessingNonexistentMapFieldsRule) checkNestedMapAccess(node *reader.RichNode, filepath string) *Finding {
	if !r.CheckNestedAccess {
		return nil
	}

	if r.isNestedKeywordAccess(node, 0) >= r.MinNestingLevel {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Deeply nested map access detected without safety checks. Consider using get-in with defaults or validating intermediate values.",
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

func (r *AccessingNonexistentMapFieldsRule) isNestedKeywordAccess(node *reader.RichNode, depth int) int {
	if node.Type == reader.NodeList && len(node.Children) == 2 {
		firstChild := node.Children[0]
		secondChild := node.Children[1]

		if firstChild.Type == reader.NodeKeyword {

			if secondChild.Type == reader.NodeList {
				return 1 + r.isNestedKeywordAccess(secondChild, depth+1)
			}

			if secondChild.Type == reader.NodeSymbol {
				return 1
			}
		}
	}

	return 0
}

func (r *AccessingNonexistentMapFieldsRule) isCommonSafePattern(keyword, mapVar string) bool {

	safeKeywords := []string{":id", ":type", ":status", ":name"}
	for _, safe := range safeKeywords {
		if keyword == safe {
			return true
		}
	}

	safeVarPatterns := []string{"validated-", "checked-", "safe-", "verified-"}
	for _, pattern := range safeVarPatterns {
		if strings.HasPrefix(mapVar, pattern) {
			return true
		}
	}

	return false
}

func (r *AccessingNonexistentMapFieldsRule) formatDirectAccessMessage(keyword, mapVar string) string {
	return fmt.Sprintf(
		"Direct map access (%s %s) without safety check detected. Consider using (get %s %s default-value) or validating the map structure first.",
		keyword, mapVar, mapVar, keyword,
	)
}

func (r *AccessingNonexistentMapFieldsRule) formatThreadingMessage(macro string, keywords []string) string {
	keywordStr := strings.Join(keywords, " ")
	return fmt.Sprintf(
		"Potentially unsafe threading macro (%s) with multiple keyword accesses [%s]. Consider using get-in with defaults or validating intermediate values.",
		macro, keywordStr,
	)
}

func init() {
	defaultRule := &AccessingNonexistentMapFieldsRule{
		Rule: Rule{
			ID:          "accessing-nonexistent-map-fields",
			Name:        "Accessing Non-Existent Map Fields",
			Description: "Detects potentially unsafe access to map fields that may not exist. Suggests using safe access patterns with defaults or validation to prevent runtime errors.",
			Severity:    SeverityWarning,
		},
		CheckDirectKeywordAccess: true,
		CheckThreadingMacros:     true,
		CheckNestedAccess:        true,
		MinNestingLevel:          2,
	}

	RegisterRule(defaultRule)
}
