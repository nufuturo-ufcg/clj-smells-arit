package rules

import "sort"

// SemanticRuleDiagnostics contains diagnostic counters for one rule. These
// counters describe analysis coverage; they never participate in finding
// promotion or suppression.
type SemanticRuleDiagnostics struct {
	ObservedCandidates int            `json:"observed_candidates"`
	ProvenFindings     int            `json:"proven_findings"`
	ContextualFindings int            `json:"contextual_findings"`
	BlockedCandidates  int            `json:"blocked_candidates"`
	BlockedByEvidence  map[string]int `json:"blocked_by_evidence,omitempty"`
}

// SemanticDiagnosticsReport is the stable, serializable form of diagnostics.
// Rules that do not produce a candidate are intentionally absent; this keeps
// the report about observed analysis paths rather than inventing coverage.
type SemanticDiagnosticsReport struct {
	Rules map[string]SemanticRuleDiagnostics `json:"rules"`
}

func (r *SemanticDiagnosticsReport) ensureRule(ruleID string) *SemanticRuleDiagnostics {
	if r.Rules == nil {
		r.Rules = make(map[string]SemanticRuleDiagnostics)
	}
	entry := r.Rules[ruleID]
	if entry.BlockedByEvidence == nil {
		entry.BlockedByEvidence = make(map[string]int)
	}
	r.Rules[ruleID] = entry
	return &entry
}

// SemanticDiagnostics is mutable state scoped to one file analysis.
type SemanticDiagnostics struct {
	report SemanticDiagnosticsReport
}

func NewSemanticDiagnostics() *SemanticDiagnostics {
	return &SemanticDiagnostics{report: SemanticDiagnosticsReport{Rules: make(map[string]SemanticRuleDiagnostics)}}
}

func (d *SemanticDiagnostics) RecordFinding(finding *Finding) {
	if d == nil || finding == nil || finding.RuleID == "" {
		return
	}
	entry := d.report.Rules[finding.RuleID]
	entry.ObservedCandidates++
	if finding.Contextual {
		entry.ContextualFindings++
		entry.BlockedCandidates++
		if entry.BlockedByEvidence == nil {
			entry.BlockedByEvidence = make(map[string]int)
		}
		seen := make(map[string]bool, len(finding.MissingEvidence))
		for _, evidence := range finding.MissingEvidence {
			if evidence == "" || seen[evidence] {
				continue
			}
			seen[evidence] = true
			entry.BlockedByEvidence[evidence]++
		}
		if len(seen) == 0 {
			entry.BlockedByEvidence["unknown-evidence"]++
		}
	} else {
		entry.ProvenFindings++
	}
	d.report.Rules[finding.RuleID] = entry
}

func (d *SemanticDiagnostics) RecordBlocked(ruleID string, evidence ...string) {
	if d == nil || ruleID == "" {
		return
	}
	entry := d.report.Rules[ruleID]
	entry.BlockedCandidates++
	if entry.BlockedByEvidence == nil {
		entry.BlockedByEvidence = make(map[string]int)
	}
	seen := make(map[string]bool, len(evidence))
	for _, item := range evidence {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		entry.BlockedByEvidence[item]++
	}
	if len(seen) == 0 {
		entry.BlockedByEvidence["unknown-evidence"]++
	}
	d.report.Rules[ruleID] = entry
}

func (d *SemanticDiagnostics) Report() SemanticDiagnosticsReport {
	if d == nil {
		return SemanticDiagnosticsReport{Rules: map[string]SemanticRuleDiagnostics{}}
	}
	result := SemanticDiagnosticsReport{Rules: make(map[string]SemanticRuleDiagnostics, len(d.report.Rules))}
	for ruleID, entry := range d.report.Rules {
		entry.BlockedByEvidence = cloneIntMap(entry.BlockedByEvidence)
		result.Rules[ruleID] = entry
	}
	return result
}

func (r *SemanticDiagnosticsReport) Merge(other SemanticDiagnosticsReport) {
	if r == nil {
		return
	}
	if r.Rules == nil {
		r.Rules = make(map[string]SemanticRuleDiagnostics)
	}
	for ruleID, incoming := range other.Rules {
		entry := r.Rules[ruleID]
		entry.ObservedCandidates += incoming.ObservedCandidates
		entry.ProvenFindings += incoming.ProvenFindings
		entry.ContextualFindings += incoming.ContextualFindings
		entry.BlockedCandidates += incoming.BlockedCandidates
		if len(incoming.BlockedByEvidence) > 0 {
			if entry.BlockedByEvidence == nil {
				entry.BlockedByEvidence = make(map[string]int)
			}
			for evidence, count := range incoming.BlockedByEvidence {
				entry.BlockedByEvidence[evidence] += count
			}
		}
		r.Rules[ruleID] = entry
	}
}

func cloneIntMap(values map[string]int) map[string]int {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]int, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

// RecordBlockedSemanticCandidate records a rule-local candidate that could
// not be promoted because named evidence was absent. It is deliberately
// opt-in: a nil diagnostics context has no effect on normal analysis.
func RecordBlockedSemanticCandidate(context map[string]interface{}, ruleID string, evidence ...string) {
	if context == nil {
		return
	}
	diagnostics, _ := context["semantic-diagnostics"].(*SemanticDiagnostics)
	if diagnostics != nil {
		diagnostics.RecordBlocked(ruleID, evidence...)
	}
}

// SortedRuleIDs is useful to human-facing consumers that want deterministic
// iteration independent of map order.
func (r SemanticDiagnosticsReport) SortedRuleIDs() []string {
	ids := make([]string, 0, len(r.Rules))
	for ruleID := range r.Rules {
		ids = append(ids, ruleID)
	}
	sort.Strings(ids)
	return ids
}
