package rules

import "github.com/thlaurentino/arit/internal/reader"

type Severity string

const (
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
	SeverityHint    Severity = "HINT"
)

// Confidence describes how much evidence supports a finding. It is kept
// separate from severity: a contextual candidate may still be important, but
// it is not a proven issue and must not be treated as one by the default CLI.
type Confidence string

const (
	ConfidenceProven     Confidence = "proven"
	ConfidenceContextual Confidence = "contextual"
)

type Finding struct {
	RuleID           string           `json:"rule_id"`
	Message          string           `json:"message"`
	Filepath         string           `json:"filepath"`
	Location         *reader.Location `json:"location"`
	Severity         Severity         `json:"severity"`
	Confidence       Confidence       `json:"confidence,omitempty"`
	MissingEvidence  []string         `json:"missing_evidence,omitempty"`
	Generated        bool             `json:"generated,omitempty"`
	OriginLocation   *reader.Location `json:"origin_location,omitempty"`
	ASTFingerprint   string           `json:"ast_fingerprint,omitempty"`
	Tags             []string         `json:"tags,omitempty"`
	SourceRole       string           `json:"source_role,omitempty"`
	Contextual       bool             `json:"contextual,omitempty"`
	ContextualReason string           `json:"contextual_reason,omitempty"`
	Intentionality   string           `json:"intentionality,omitempty"`
}
