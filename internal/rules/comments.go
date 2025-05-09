package rules

import (
	"fmt"
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
)

type CommentsRule struct {
	Rule
	ReportTODO      bool `json:"report_todo" yaml:"report_todo"`
	ReportFIXME     bool `json:"report_fixme" yaml:"report_fixme"`
	ReportEmpty     bool `json:"report_empty" yaml:"report_empty"`
	ReportRedundant bool `json:"report_redundant" yaml:"report_redundant"`
	ReportObvious   bool `json:"report_obvious" yaml:"report_obvious"`
}

func (r *CommentsRule) Meta() Rule {
	return r.Rule
}

func isRedundantComment(comment string) bool {
	// Remove comment markers and trim spaces
	comment = strings.TrimSpace(strings.TrimPrefix(comment, ";;"))
	comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))

	// List of redundant patterns
	redundantPatterns := []string{
		"function to",
		"method to",
		"this function",
		"this method",
		"calls",
		"returns",
	}

	commentLower := strings.ToLower(comment)
	for _, pattern := range redundantPatterns {
		if strings.Contains(commentLower, pattern) {
			return true
		}
	}

	return false
}

func isObviousComment(comment string) bool {

	comment = strings.TrimSpace(strings.TrimPrefix(comment, ";;"))
	comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))

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
	}

	for _, pattern := range obviousPatterns {
		if strings.Contains(strings.ToLower(comment), pattern) {
			return true
		}
	}

	return false
}

func (r *CommentsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	if node.Type != reader.NodeComment {
		return nil
	}

	commentText := strings.TrimSpace(node.Value)
	currentLocation := node.Location

	if r.ReportEmpty && commentText == "" {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Empty or whitespace-only comment found. Consider removing it.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: r.Severity,
		}
	}

	if r.ReportTODO && strings.Contains(strings.ToUpper(commentText), "TODO") {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("TODO comment found: %s. Consider creating an issue/task instead.", node.Value),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	if r.ReportFIXME && strings.Contains(strings.ToUpper(commentText), "FIXME") {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("FIXME comment found: %s. Consider creating an issue/task instead.", node.Value),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityWarning,
		}
	}

	if r.ReportRedundant && isRedundantComment(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Redundant comment that just describes what the code does: %s. Consider removing it or explaining 'why' instead of 'what'.", node.Value),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	if r.ReportObvious && isObviousComment(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Comment states the obvious: %s. Consider removing it or providing more meaningful information.", node.Value),
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
			Name:        "Comment Analysis",
			Description: "Analyzes comments for common issues like TODOs, FIXMEs, redundancy, and obvious statements. Comments should explain 'why' not 'what'. Code should be self-documenting through good naming and structure. Use docstrings for API documentation.",
			Severity:    SeverityInfo,
		},
		ReportTODO:      true,
		ReportFIXME:     true,
		ReportEmpty:     true,
		ReportRedundant: true,
		ReportObvious:   true,
	}

	RegisterRule(defaultRule)
}
