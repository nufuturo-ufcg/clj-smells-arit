package reporter

import (
	"io"
	"sort"

	"github.com/thlaurentino/arit/internal/rules"
)

type ReportFormat string

const (
	FormatJSON        ReportFormat = "json"
	FormatJSONSnippet ReportFormat = "json-snippet"
	FormatText        ReportFormat = "text"
	FormatHTML        ReportFormat = "html"
	FormatMarkdown    ReportFormat = "markdown"
	FormatSummary     ReportFormat = "summary"
	FormatCSV         ReportFormat = "csv"
	FormatSARIF       ReportFormat = "sarif"
)

type Reporter interface {
	Report(findings []*rules.Finding, writer io.Writer) error
}

type SummaryItem struct {
	RuleID string
	Count  int
}

type ContextualSummaryItem struct {
	RuleID        string
	NonContextual int
	Contextual    int
}

func FilterContextualFindings(findings []*rules.Finding, include bool) []*rules.Finding {
	if include {
		return findings
	}
	filtered := make([]*rules.Finding, 0, len(findings))
	for _, finding := range findings {
		if !rules.IsContextualFinding(finding) {
			filtered = append(filtered, finding)
		}
	}
	return filtered
}

func getContextualSummary(findings []*rules.Finding) []ContextualSummaryItem {
	counts := make(map[string]*ContextualSummaryItem)
	for _, finding := range findings {
		item := counts[finding.RuleID]
		if item == nil {
			item = &ContextualSummaryItem{RuleID: finding.RuleID}
			counts[finding.RuleID] = item
		}
		if rules.IsContextualFinding(finding) {
			item.Contextual++
		} else {
			item.NonContextual++
		}
	}
	items := make([]ContextualSummaryItem, 0, len(counts))
	for _, item := range counts {
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RuleID < items[j].RuleID })
	return items
}

func getSortedSummary(findings []*rules.Finding) []SummaryItem {
	summary := make(map[string]int)
	for _, finding := range findings {
		summary[finding.RuleID]++
	}

	var ruleIDs []string
	for ruleID := range summary {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)

	var summaryItems []SummaryItem
	for _, ruleID := range ruleIDs {
		summaryItems = append(summaryItems, SummaryItem{
			RuleID: ruleID,
			Count:  summary[ruleID],
		})
	}

	return summaryItems
}

func NewReporter(format ReportFormat) Reporter {
	switch format {
	case FormatJSON:
		return &JSONReporter{}
	case FormatJSONSnippet, "json-extended", "jsonsnippet", "json_snippet":
		return &JSONSnippetReporter{}
	case FormatText:
		return &TextReporter{}
	case FormatHTML:
		return &HTMLReporter{}
	case FormatMarkdown:
		return &MarkdownReporter{}
	case FormatSummary:
		return &SummaryReporter{}
	case FormatCSV:
		return &CSVReporter{}
	case FormatSARIF:
		return &SARIFReporter{}
	default:
		return nil
	}
}
