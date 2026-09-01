package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thlaurentino/arit/internal/rules"
)

func writeSemanticDiagnostics(path string, report rules.SemanticDiagnosticsReport) error {
	if path == "-" {
		return fmt.Errorf("semantic diagnostics path must be a file, not stdout")
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
