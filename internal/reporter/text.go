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

type SummaryReporter struct{}

func (sr *SummaryReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, "No issues found.")
		return err
	}

	_, err := fmt.Fprintf(writer, "Total findings: %d\n\n", len(findings))
	if err != nil {
		return err
	}

	summaryItems := getSortedSummary(findings)
	if len(summaryItems) > 0 {
		_, _ = fmt.Fprintln(writer, "Smell Summary:")
		for _, item := range summaryItems {
			_, _ = fmt.Fprintf(writer, "- %s: %d\n", item.RuleID, item.Count)
		}
	}

	return nil
}
