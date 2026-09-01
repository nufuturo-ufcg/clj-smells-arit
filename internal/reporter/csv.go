package reporter

import (
	"bufio"
	"fmt"
	"io"
	"sort"

	"github.com/thlaurentino/arit/internal/rules"
)

type CSVReporter struct{}

func (cr *CSVReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	w := bufio.NewWriter(writer)

	_, err := w.WriteString("RuleID,Count\n")
	if err != nil {
		return err
	}

	summary := make(map[string]int)
	for _, f := range findings {
		summary[f.RuleID]++
	}

	var ruleIDs []string
	for ruleID := range summary {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)

	for _, ruleID := range ruleIDs {
		line := fmt.Sprintf("%s,%d\n", ruleID, summary[ruleID])
		_, err := w.WriteString(line)
		if err != nil {
			return err
		}
	}

	return w.Flush()
}
