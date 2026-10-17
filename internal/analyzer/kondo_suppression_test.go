package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thlaurentino/arit/internal/config"
	_ "github.com/thlaurentino/arit/internal/rules/clojurespecific"
)

// clj-kondo suppresses with discarded forms, not comments. The bare keyword
// ignores every linter; a map with a linter list ignores only those linters.
func TestKondoDiscardDirectiveSuppressesNextForm(t *testing.T) {
	source := "(ns k)\n\n(defn f []\n  #_:clj-kondo/ignore\n  (def a 1)\n  #_{:clj-kondo/ignore [:inline-def]}\n  (def b 2)\n  (def c 3))\n"
	path := filepath.Join(t.TempDir(), "k.clj")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := AnalyzeFile(path, &config.Config{EnabledRules: map[string]bool{}, EnabledGroups: map[string]bool{}, RuleConfig: map[string]config.RuleSettings{}})
	if err != nil {
		t.Fatal(err)
	}
	lines := map[int]bool{}
	for _, finding := range result.Findings {
		if finding.RuleID == "immutability-violation" {
			lines[finding.Location.StartLine] = true
		}
	}
	if lines[5] {
		t.Errorf("finding after #_:clj-kondo/ignore was not suppressed: %v", lines)
	}
	// A linter-specific directive names clj-kondo linters, not ARIT rules.
	if !lines[7] || !lines[8] {
		t.Errorf("defs on lines 7 and 8 must still be reported: %v", lines)
	}
}
