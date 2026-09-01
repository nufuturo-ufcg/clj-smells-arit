package reporter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thlaurentino/arit/internal/rules"
)

type JSONReporter struct{}

func (jr *JSONReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(findings)
	if err != nil {
		return fmt.Errorf("error encoding findings to JSON: %w", err)
	}
	return nil
}

type JSONSnippetReporter struct{}

type JSONSnippetFinding struct {
	*rules.Finding
	CodeSnippet string `json:"code_snippet"`
}

func getProblemCodeText(finding *rules.Finding) string {
	const contextLines = 4

	if finding.Location == nil || finding.Filepath == "" {
		return ""
	}

	file, err := os.Open(finding.Filepath)
	if err != nil {
		return ""
	}
	defer file.Close()

	var outputLines []string
	scanner := bufio.NewScanner(file)
	currentLine := 1

	startContextLine := finding.Location.StartLine - contextLines
	if startContextLine < 1 {
		startContextLine = 1
	}
	endContextLine := finding.Location.EndLine + contextLines

	for scanner.Scan() {
		if currentLine >= startContextLine && currentLine <= endContextLine {
			lineText := scanner.Text()

			isWithinFindingRange := currentLine >= finding.Location.StartLine && currentLine <= finding.Location.EndLine

			if strings.TrimSpace(lineText) == "" && !isWithinFindingRange {
				currentLine++
				continue
			}

			lineWithNumber := fmt.Sprintf("%5d: %s", currentLine, lineText)
			outputLines = append(outputLines, lineWithNumber)
		}
		if currentLine > endContextLine {
			break
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	if len(outputLines) == 0 {
		return "// No code found at location or file is empty."
	}
	return strings.Join(outputLines, "\n")
}

func (jsr *JSONSnippetReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	snippetFindings := make([]JSONSnippetFinding, len(findings))
	for i, f := range findings {
		snippetFindings[i] = JSONSnippetFinding{
			Finding:     f,
			CodeSnippet: getProblemCodeText(f),
		}
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(snippetFindings)
	if err != nil {
		return fmt.Errorf("error encoding findings with snippets to JSON: %w", err)
	}
	return nil
}
