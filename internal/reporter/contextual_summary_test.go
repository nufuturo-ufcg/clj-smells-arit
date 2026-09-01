package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/thlaurentino/arit/internal/rules"
)

func TestFilterContextualFindings(t *testing.T) {
	nonContextual := &rules.Finding{RuleID: "real", Severity: rules.SeverityWarning}
	contextual := &rules.Finding{RuleID: "context", Severity: rules.SeverityHint, Contextual: true}
	findings := []*rules.Finding{nonContextual, contextual}

	filtered := FilterContextualFindings(findings, false)
	if len(filtered) != 1 || filtered[0] != nonContextual {
		t.Fatalf("expected only the non-contextual finding, got %#v", filtered)
	}
	if included := FilterContextualFindings(findings, true); len(included) != len(findings) {
		t.Fatalf("expected all findings with contextual inclusion, got %d", len(included))
	}
}

func TestSummarySeparatesContextualFindings(t *testing.T) {
	nonContextual := &rules.Finding{RuleID: "real", Severity: rules.SeverityWarning}
	contextual := &rules.Finding{RuleID: "context", Severity: rules.SeverityHint, Contextual: true}
	findings := []*rules.Finding{nonContextual, contextual}

	var output bytes.Buffer
	reporter := &SummaryReporter{}
	reporter.SetContextualSummary(findings, true)
	if err := reporter.Report(findings, &output); err != nil {
		t.Fatalf("report failed: %v", err)
	}

	for _, expected := range []string{
		"Total findings displayed: 2",
		"Real (non-contextual) findings: 1",
		"Possible contextual findings: 1",
		"- context: 1 (non-contextual: 0; possible contextual: 1)",
		"- real: 1 (non-contextual: 1; possible contextual: 0)",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("expected summary to contain %q; output was:\n%s", expected, output.String())
		}
	}
}

func TestSummaryReportsHiddenContextualFindings(t *testing.T) {
	contextual := &rules.Finding{RuleID: "context", Severity: rules.SeverityHint, Contextual: true}

	var output bytes.Buffer
	reporter := &SummaryReporter{}
	reporter.SetContextualSummary([]*rules.Finding{contextual}, false)
	if err := reporter.Report(nil, &output); err != nil {
		t.Fatalf("report failed: %v", err)
	}

	expected := "No real (non-contextual) issues found. Possible contextual findings hidden: 1. Use --include-contextual to display them."
	if strings.TrimSpace(output.String()) != expected {
		t.Fatalf("expected hidden contextual summary %q, got %q", expected, strings.TrimSpace(output.String()))
	}
}
