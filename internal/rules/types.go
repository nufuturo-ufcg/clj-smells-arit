package rules

import "github.com/thlaurentino/arit/internal/reader"

type Severity string

const (
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
	SeverityHint    Severity = "HINT"
)

type Finding struct {
	RuleID   string           `json:"rule_id"`
	Message  string           `json:"message"`
	Filepath string           `json:"filepath"`
	Location *reader.Location `json:"location"`
	Severity Severity         `json:"severity"`
}
