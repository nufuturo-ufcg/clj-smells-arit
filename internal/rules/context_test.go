package rules

import (
	"testing"

	"github.com/thlaurentino/arit/internal/reader"
)

func TestSourceRoleDoesNotMakeFindingContextual(t *testing.T) {
	finding := &Finding{Severity: SeverityHint}
	MarkContextualFinding(finding)

	if finding.Contextual {
		t.Fatal("a source role must not be used as contextuality")
	}
	if finding.Confidence != ConfidenceProven {
		t.Fatalf("expected proven confidence, got %q", finding.Confidence)
	}

	fixtureContext := map[string]interface{}{"file-role": "fixture"}
	if got := ContextualSeverity(fixtureContext, SeverityWarning); got != SeverityHint {
		t.Fatalf("expected fixture severity to remain a hint, got %q", got)
	}
}

func TestSetContextualFindingSetsConfidenceAndReason(t *testing.T) {
	finding := SetContextualFinding(&Finding{}, "requires an external contract")

	if !finding.Contextual {
		t.Fatal("expected contextual finding")
	}
	if finding.Confidence != ConfidenceContextual {
		t.Fatalf("expected contextual confidence, got %q", finding.Confidence)
	}
	if finding.ContextualReason != "requires an external contract" {
		t.Fatalf("unexpected contextual reason %q", finding.ContextualReason)
	}
	if len(finding.MissingEvidence) != 1 || finding.MissingEvidence[0] != "external-contract" {
		t.Fatalf("unexpected missing evidence %#v", finding.MissingEvidence)
	}
}

func TestSetContextualFindingWithEvidenceUsesSpecificEvidence(t *testing.T) {
	finding := SetContextualFindingWithEvidence(&Finding{}, "requires type information", "collection-type", "return-contract")

	if len(finding.MissingEvidence) != 2 || finding.MissingEvidence[0] != "collection-type" || finding.MissingEvidence[1] != "return-contract" {
		t.Fatalf("unexpected specific evidence %#v", finding.MissingEvidence)
	}
}

func TestBuilderContextualWithEvidenceUsesSpecificEvidence(t *testing.T) {
	rule := NewRule("test-contextual-evidence").
		When(func(_ *reader.RichNode, _ map[string]interface{}, _ string) bool { return true }).
		Message("candidate").
		ContextualWithEvidence("requires a contract", "api-contract", "lifecycle-contract").
		Register()

	finding := rule.Check(&reader.RichNode{}, nil, "test.clj")
	if finding == nil || finding.Confidence != ConfidenceContextual {
		t.Fatalf("expected contextual finding, got %#v", finding)
	}
	if len(finding.MissingEvidence) != 2 || finding.MissingEvidence[0] != "api-contract" || finding.MissingEvidence[1] != "lifecycle-contract" {
		t.Fatalf("unexpected builder evidence %#v", finding.MissingEvidence)
	}
}
