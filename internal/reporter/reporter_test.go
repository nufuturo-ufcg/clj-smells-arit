package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

func TestJSONSnippetReporter(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.clj")
	content := "(ns test.core)\n(defn foo []\n  (println \"hello\"))\n"
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	finding := &rules.Finding{
		RuleID:   "test-rule",
		Message:  "Test issue",
		Filepath: testFile,
		Location: &reader.Location{
			StartLine:   2,
			StartColumn: 1,
			EndLine:     3,
			EndColumn:   20,
		},
		Severity: rules.SeverityWarning,
	}

	rep := NewReporter(FormatJSONSnippet)
	assert.NotNil(t, rep)

	var buf bytes.Buffer
	err = rep.Report([]*rules.Finding{finding}, &buf)
	assert.NoError(t, err)

	var results []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	item := results[0]
	assert.Equal(t, "test-rule", item["rule_id"])
	assert.Equal(t, "Test issue", item["message"])
	assert.Equal(t, testFile, item["filepath"])
	assert.Contains(t, item, "code_snippet")

	snippet, ok := item["code_snippet"].(string)
	assert.True(t, ok)
	assert.Contains(t, snippet, "defn foo")
}
