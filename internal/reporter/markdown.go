package reporter

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/thlaurentino/arit/internal/rules"
)

type MarkdownReporter struct{}

func (md *MarkdownReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, "No issues found.")
		return err
	}

	w := bufio.NewWriter(writer)

	_, err := w.WriteString("| Severity | Rule ID | Message | File & Location |\n")
	if err != nil {
		return err
	}
	_, err = w.WriteString("|---|---|---|---|\n")
	if err != nil {
		return err
	}

	for _, finding := range findings {
		var loc string
		if finding.Location != nil {
			loc = fmt.Sprintf("`%s:%d:%d`", finding.Filepath, finding.Location.StartLine, finding.Location.StartColumn)
		} else {
			loc = fmt.Sprintf("`%s`", finding.Filepath)
		}

		message := strings.ReplaceAll(finding.Message, "|", "\\|")

		line := fmt.Sprintf("| %s | %s | %s | %s |\n",
			finding.Severity,
			finding.RuleID,
			message,
			loc)

		_, err := w.WriteString(line)
		if err != nil {
			return err
		}
	}

	summaryItems := getSortedSummary(findings)
	if len(summaryItems) > 0 {
		_, _ = w.WriteString("\n### Smell Summary\n\n")
		for _, item := range summaryItems {
			_, _ = w.WriteString(fmt.Sprintf("- **%s**: %d\n", item.RuleID, item.Count))
		}
	}

	return w.Flush()
}
