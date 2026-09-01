package reporter

import (
	"fmt"
	"io"

	"github.com/thlaurentino/arit/internal/rules"
)

type TextReporter struct{}

func (tr *TextReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, "No issues found.")
		return err
	}

	for _, finding := range findings {
		loc := "(file-level)"
		if finding.Location != nil {
			loc = fmt.Sprintf("%s:%d:%d", finding.Filepath, finding.Location.StartLine, finding.Location.StartColumn)
		} else {
			loc = finding.Filepath
		}

		line := fmt.Sprintf("[%s] %s: %s [%s]\n",
			finding.Severity,
			finding.RuleID,
			finding.Message,
			loc)

		_, err := fmt.Fprint(writer, line)
		if err != nil {
			return fmt.Errorf("error writing finding: %w", err)
		}
	}

	summaryItems := getSortedSummary(findings)
	if len(summaryItems) > 0 {
		_, _ = fmt.Fprintln(writer, "\n---")
		_, _ = fmt.Fprintln(writer, "Smell Summary:")
		for _, item := range summaryItems {
			_, _ = fmt.Fprintf(writer, "- %s: %d\n", item.RuleID, item.Count)
		}
	}

	return nil
}

type SummaryReporter struct {
	allFindings       []*rules.Finding
	includeContextual bool
}

func (sr *SummaryReporter) SetContextualSummary(allFindings []*rules.Finding, includeContextual bool) {
	sr.allFindings = allFindings
	sr.includeContextual = includeContextual
}

func (sr *SummaryReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	allFindings := sr.allFindings
	if allFindings == nil {
		allFindings = findings
	}
	contextualSummary := getContextualSummary(allFindings)
	nonContextual, contextual := 0, 0
	for _, item := range contextualSummary {
		nonContextual += item.NonContextual
		contextual += item.Contextual
	}
	if len(findings) == 0 {
		if contextual > 0 && !sr.includeContextual {
			_, err := fmt.Fprintf(writer, "No real (non-contextual) issues found. Possible contextual findings hidden: %d. Use --include-contextual to display them.\n", contextual)
			return err
		}
		_, err := fmt.Fprintln(writer, "No issues found.")
		return err
	}

	_, err := fmt.Fprintf(writer, "Total findings displayed: %d\n", len(findings))
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(writer, "Real (non-contextual) findings: %d\n", nonContextual)
	_, _ = fmt.Fprintf(writer, "Possible contextual findings: %d", contextual)
	if contextual > 0 && !sr.includeContextual {
		_, _ = fmt.Fprint(writer, " (hidden; use --include-contextual)")
	}
	_, _ = fmt.Fprintln(writer)

	if len(contextualSummary) > 0 {
		_, _ = fmt.Fprintln(writer, "Smell Summary:")
		for _, item := range contextualSummary {
			total := item.NonContextual + item.Contextual
			if sr.includeContextual {
				_, _ = fmt.Fprintf(writer, "- %s: %d (non-contextual: %d; possible contextual: %d)\n", item.RuleID, total, item.NonContextual, item.Contextual)
			} else if item.NonContextual > 0 {
				_, _ = fmt.Fprintf(writer, "- %s: %d\n", item.RuleID, item.NonContextual)
			}
		}
	}

	return nil
}
