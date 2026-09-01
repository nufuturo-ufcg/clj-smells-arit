package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/thlaurentino/arit/internal/rules"
)

type SARIFReporter struct{}

func (sr *SARIFReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	sarif := map[string]interface{}{
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		"version": "2.1.0",
		"runs": []map[string]interface{}{
			{
				"tool": map[string]interface{}{
					"driver": map[string]interface{}{
						"name":           "ARIT",
						"informationUri": "https://github.com/thlaurentino/arit",
						"rules":          buildSarifRules(findings),
					},
				},
				"results": buildSarifResults(findings),
			},
		},
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(sarif); err != nil {
		return fmt.Errorf("error encoding findings to SARIF: %w", err)
	}

	return nil
}

func buildSarifRules(findings []*rules.Finding) []map[string]interface{} {
	ruleSet := make(map[string]bool)
	var sarifRules []map[string]interface{}

	for _, f := range findings {
		if !ruleSet[f.RuleID] {
			ruleSet[f.RuleID] = true
			sarifRules = append(sarifRules, map[string]interface{}{
				"id": f.RuleID,
				"name": f.RuleID,
				"shortDescription": map[string]interface{}{
					"text": f.RuleID,
				},
			})
		}
	}
	return sarifRules
}

func buildSarifResults(findings []*rules.Finding) []map[string]interface{} {
	var results []map[string]interface{}

	for _, f := range findings {
		level := "note"
		switch string(f.Severity) {
		case "ERROR":
			level = "error"
		case "WARNING":
			level = "warning"
		case "INFO", "HINT":
			level = "note"
		}

		result := map[string]interface{}{
			"ruleId": f.RuleID,
			"level":  level,
			"message": map[string]interface{}{
				"text": f.Message,
			},
		}

		if f.Location != nil && f.Filepath != "" {
			// Convert backslashes to forward slashes for cross-platform URI
			uri := strings.ReplaceAll(f.Filepath, "\\", "/")
			
			region := map[string]interface{}{
				"startLine":   f.Location.StartLine,
				"startColumn": f.Location.StartColumn,
			}
			
			if f.Location.EndLine > 0 {
				region["endLine"] = f.Location.EndLine
			}
			if f.Location.EndColumn > 0 {
				region["endColumn"] = f.Location.EndColumn
			}

			result["locations"] = []map[string]interface{}{
				{
					"physicalLocation": map[string]interface{}{
						"artifactLocation": map[string]interface{}{
							"uri": uri,
						},
						"region": region,
					},
				},
			}
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		return []map[string]interface{}{}
	}

	return results
}
