package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type CommentType int

const (
	CommentEmpty CommentType = iota
	CommentTODO
	CommentFIXME
	CommentRedundant
	CommentObvious
	CommentDeodorant
	CommentCommentedCode
	CommentDocstring
)

type CommentsRule struct {
	Rule
	ReportTODO          bool `json:"report_todo" yaml:"report_todo"`
	ReportFIXME         bool `json:"report_fixme" yaml:"report_fixme"`
	ReportEmpty         bool `json:"report_empty" yaml:"report_empty"`
	ReportRedundant     bool `json:"report_redundant" yaml:"report_redundant"`
	ReportObvious       bool `json:"report_obvious" yaml:"report_obvious"`
	ReportDeodorant     bool `json:"report_deodorant" yaml:"report_deodorant"`
	ReportCommentedCode bool `json:"report_commented_code" yaml:"report_commented_code"`
	ReportDocstring     bool `json:"report_docstring" yaml:"report_docstring"`
	MinCommentLength    int  `json:"min_comment_length" yaml:"min_comment_length"`
	MaxCommentLength    int  `json:"max_comment_length" yaml:"max_comment_length"`
}

func (r *CommentsRule) Meta() Rule {
	return r.Rule
}

func isClojureCode(comment string) bool {

	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))

	codePatterns := []string{
		`^\s*\(`,
		`^\s*\[`,
		`^\s*\{`,
		`^\s*def[a-z-]*\s`,
		`^\s*let\s*\[`,
		`^\s*if\s*\(`,
		`^\s*when\s*\(`,
		`^\s*cond\s*$`,
		`^\s*case\s*\(`,
		`^\s*loop\s*\[`,
		`^\s*recur\s*\(`,
		`^\s*->\s*\(`,
		`^\s*->>\s*\(`,
		`^\s*require\s*\[`,
		`^\s*import\s*\[`,
		`^\s*ns\s+[a-z]`,
	}

	for _, pattern := range codePatterns {
		if matched, _ := regexp.MatchString(pattern, cleaned); matched {
			return true
		}
	}

	parenCount := strings.Count(cleaned, "(") + strings.Count(cleaned, "[") + strings.Count(cleaned, "{")
	return parenCount >= 2
}

func isRedundantComment(comment string, nextNode *reader.RichNode) bool {

	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))
	commentLower := strings.ToLower(cleaned)

	redundantPatterns := []string{
		"function to",
		"method to",
		"this function",
		"this method",
		"calls",
		"returns",
		"defines",
		"creates",
		"sets",
		"gets",
		"increments",
		"decrements",
	}

	for _, pattern := range redundantPatterns {
		if strings.Contains(commentLower, pattern) {
			return true
		}
	}

	if nextNode != nil && nextNode.Type == reader.NodeList {
		if len(nextNode.Children) > 0 {
			firstChild := nextNode.Children[0]
			if firstChild.Type == reader.NodeSymbol {
				symbol := strings.ToLower(firstChild.Value)

				if strings.HasPrefix(symbol, "def") {
					if strings.Contains(commentLower, "define") ||
						strings.Contains(commentLower, "definition") ||
						strings.Contains(commentLower, symbol) {
						return true
					}
				}

				if symbol == "let" && strings.Contains(commentLower, "let") {
					return true
				}

				if (symbol == "if" || symbol == "when") &&
					(strings.Contains(commentLower, "if") || strings.Contains(commentLower, "when")) {
					return true
				}
			}
		}
	}

	return false
}

func isObviousComment(comment string) bool {

	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))
	commentLower := strings.ToLower(cleaned)

	obviousPatterns := []string{
		"initialize",
		"start",
		"end",
		"validate",
		"check",
		"store",
		"save",
		"transform",
		"convert",
		"process",
		"handle",
		"manage",
		"update",
		"delete",
		"create",
		"add",
		"remove",
		"get",
		"set",
		"main function",
		"helper function",
		"utility function",
	}

	for _, pattern := range obviousPatterns {
		if strings.Contains(commentLower, pattern) {
			return true
		}
	}

	return false
}

func isDeodorantComment(comment string) bool {

	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))
	commentLower := strings.ToLower(cleaned)

	deodorantPatterns := []string{
		"hack",
		"workaround",
		"temporary",
		"quick fix",
		"dirty",
		"ugly",
		"bad",
		"terrible",
		"awful",
		"sorry",
		"apologize",
		"don't ask",
		"magic number",
		"hardcoded",
		"hard coded",
		"kludge",
		"bodge",
		"duct tape",
	}

	for _, pattern := range deodorantPatterns {
		if strings.Contains(commentLower, pattern) {
			return true
		}
	}

	return false
}

func (r *CommentsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type == reader.NodeComment {
		return r.checkComment(node, context, filepath)
	}

	return nil
}

func (r *CommentsRule) checkComment(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	commentText := strings.TrimSpace(node.Value)
	currentLocation := node.Location

	if r.ReportEmpty && (commentText == "" || commentText == ";" || commentText == ";;") {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Empty comment found. Consider removing it.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	cleanedComment := strings.TrimSpace(strings.TrimPrefix(commentText, ";"))
	cleanedComment = strings.TrimSpace(strings.TrimPrefix(cleanedComment, ";;"))

	if r.MinCommentLength > 0 && len(cleanedComment) < r.MinCommentLength && len(cleanedComment) > 0 {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Comment is too short (%d chars). Consider providing more context or removing it.", len(cleanedComment)),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	if r.MaxCommentLength > 0 && len(cleanedComment) > r.MaxCommentLength {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Comment is too long (%d chars). Consider breaking it into multiple lines or using docstrings.", len(cleanedComment)),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	commentUpper := strings.ToUpper(commentText)
	if r.ReportTODO && strings.Contains(commentUpper, "TODO") {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("TODO comment found: %s. Consider creating an issue/task instead.", commentText),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	if r.ReportFIXME && strings.Contains(commentUpper, "FIXME") {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("FIXME comment found: %s. Consider creating an issue/task instead.", commentText),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityWarning,
		}
	}

	if r.ReportCommentedCode && isClojureCode(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Commented-out code found. Consider removing it or using version control instead.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityWarning,
		}
	}

	if r.ReportDeodorant && isDeodorantComment(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Deodorant comment found. This comment suggests the code needs refactoring rather than explanation.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityWarning,
		}
	}

	var nextNode *reader.RichNode
	if parent := context["parent"]; parent != nil {
		if parentNode, ok := parent.(*reader.RichNode); ok {
			for i, child := range parentNode.Children {
				if child == node && i+1 < len(parentNode.Children) {
					nextNode = parentNode.Children[i+1]
					break
				}
			}
		}
	}

	if r.ReportRedundant && isRedundantComment(commentText, nextNode) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Redundant comment that just describes what the code does. Consider explaining 'why' instead of 'what', or remove it.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	if r.ReportObvious && isObviousComment(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Comment states the obvious. Consider removing it or providing more meaningful information.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	return nil
}

func init() {
	defaultRule := &CommentsRule{
		Rule: Rule{
			ID:          "comments",
			Name:        "Comment Quality Analysis",
			Description: "Analyzes comments for quality issues in Clojure code. Detects redundant, obvious, deodorant comments, commented-out code, and missing docstrings. Promotes self-documenting code through good naming and structure. Comments should explain 'why' not 'what'.",
			Severity:    SeverityInfo,
		},
		ReportTODO:          true,
		ReportFIXME:         true,
		ReportEmpty:         true,
		ReportRedundant:     true,
		ReportObvious:       true,
		ReportDeodorant:     true,
		ReportCommentedCode: true,
		ReportDocstring:     false,
		MinCommentLength:    5,
		MaxCommentLength:    120,
	}

	RegisterRule(defaultRule)
}
