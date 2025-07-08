package reporter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	io "io"
	"os"
	"sort"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

type ReportFormat string

const (
	FormatJSON     ReportFormat = "json"
	FormatText     ReportFormat = "text"
	FormatHTML     ReportFormat = "html"
	FormatMarkdown ReportFormat = "markdown"
)

type Reporter interface {
	Report(findings []*rules.Finding, writer io.Writer) error
}

type SummaryItem struct {
	RuleID string
	Count  int
}

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

type TextReporter struct{}

func (tr *TextReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, "No issues found.")
		return err
	}

	for _, finding := range findings {
		loc := "(file-level)"
		if finding.Location != nil {
			loc = fmt.Sprintf("%s:%d:%d", finding.Filepath, finding.Location.StartLine, finding.Location.StartColumn)
		} else {
			loc = finding.Filepath
		}

		line := fmt.Sprintf("[%s] %s: %s [%s]\n",
			finding.Severity,
			finding.RuleID,
			finding.Message,
			loc)

		_, err := fmt.Fprint(writer, line)
		if err != nil {
			return fmt.Errorf("error writing finding: %w", err)
		}
	}

	summaryItems := getSortedSummary(findings)
	if len(summaryItems) > 0 {
		_, _ = fmt.Fprintln(writer, "\n---")
		_, _ = fmt.Fprintln(writer, "Smell Summary:")
		for _, item := range summaryItems {
			_, _ = fmt.Fprintf(writer, "- %s: %d\n", item.RuleID, item.Count)
		}
	}

	return nil
}

type HTMLReporter struct{}

type HTMLReportData struct {
	TotalFilesAnalyzed int
	TotalFindings      int
	Findings           []*EnrichedFinding
	SummaryItems       []SummaryItem
}

type EnrichedFinding struct {
	*rules.Finding
	ProblemCode  template.HTML
	FormattedLoc string
}

func getProblemCode(finding *rules.Finding) (template.HTML, error) {
	const contextLines = 4

	if finding.Location == nil || finding.Filepath == "" {
		return "", nil
	}

	file, err := os.Open(finding.Filepath)
	if err != nil {
		return "", fmt.Errorf("error opening file %s: %w", finding.Filepath, err)
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

			escapedLineText := template.HTMLEscapeString(lineText)
			lineWithNumber := fmt.Sprintf("%5d: %s", currentLine, escapedLineText)

			if currentLine >= finding.Location.StartLine && currentLine <= finding.Location.EndLine {
				outputLines = append(outputLines, fmt.Sprintf("<span class=\"highlight-finding\">%s</span>", lineWithNumber))
			} else {
				outputLines = append(outputLines, lineWithNumber)
			}
		}
		if currentLine > endContextLine {
			break
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file %s: %w", finding.Filepath, err)
	}

	if len(outputLines) == 0 {
		return template.HTML("// No code found at location or file is empty."), nil
	}
	return template.HTML(strings.Join(outputLines, "<br>")), nil
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
<title>Arit Analysis Report</title>
<style>
  body { font-family: sans-serif; margin: 20px; }
  table { border-collapse: collapse; width: 100%; margin-top: 1em; table-layout: fixed; }
  th, td { border: 1px solid #ddd; padding: 10px 8px; text-align: left; word-wrap: break-word; }
  th { background-color: #f2f2f2; font-weight: bold; }
  .col-counter { width: 5%; }
  .col-severity { width: 10%; }
  .col-rule-id { width: 15%; }
  .col-message { width: 30%; }
  .col-file-loc { width: 20%; }
  .col-code { width: 30%; }
  tbody tr:nth-child(even) { background-color: #f9f9f9; }
  tbody tr:hover { background-color: #f1f1f1; }
  .summary { margin-bottom: 20px; padding: 10px; background-color: #eef; border: 1px solid #ccf; }
  .summary p { margin: 5px 0; }
  .severity-ERROR { color: red; font-weight: bold; }
  .severity-WARNING { color: orange; }
  .severity-INFO { color: blue; }
  .location { font-family: monospace; }
  .code-snippet { background-color: #f8f8f8; border: 1px solid #eee; padding: 10px; margin-top: 5px; white-space: pre-wrap; font-family: monospace; }
  .finding-details td { vertical-align: top; }
  .highlight-finding { 
    background-color: #fff8dc;
    display: block;
    margin: -1px;
    padding: 1px; 
  }
  .finding-message-cell { width: 30%; }
</style>
</head>
<body>

<h1>Arit Analysis Report</h1>

<div class="summary">
  <p><strong>Total Files Analyzed:</strong> {{.TotalFilesAnalyzed}}</p>
  <p><strong>Total Findings:</strong> {{.TotalFindings}}</p>
</div>

{{if .SummaryItems}}
<h2>Smell Summary</h2>
<table id="summary-table">
  <thead>
    <tr>
      <th>Smell Type (Rule ID)</th>
      <th>Count</th>
    </tr>
  </thead>
  <tbody>
    {{range .SummaryItems}}
    <tr>
      <td>{{.RuleID}}</td>
      <td>{{.Count}}</td>
    </tr>
    {{end}}
  </tbody>
</table>
{{end}}

{{if .Findings}}
  <h2>All Findings</h2>
  <table id="findings-table">
    <thead>
      <tr>
        <th class="col-counter">#</th>
        <th class="col-severity">Severity</th>
        <th class="col-rule-id">Rule ID</th>
        <th class="col-message">Message</th>
        <th class="col-file-loc">File & Location</th>
        <th class="col-code">Problematic Code</th>
      </tr>
    </thead>
    <tbody>
      {{range $i, $finding := .Findings}}
      <tr class="finding-details">
        <td>{{add $i 1}}</td>
        <td class="severity-{{$finding.Severity}}">{{$finding.Severity}}</td>
        <td>{{$finding.RuleID}}</td>
        <td class="finding-message-cell">{{$finding.Message}}</td>
        <td>
          {{$finding.Filepath}}<br>
          <span class="location">{{$finding.FormattedLoc}}</span>
        </td>
        <td class="finding-code-cell">
          {{if $finding.ProblemCode}}
          <pre class="code-snippet"><code>{{$finding.ProblemCode}}</code></pre>
          {{else}}
          <p>N/A</p>
          {{end}}
        </td>
      </tr>
      {{end}}
    </tbody>
  </table>
{{else}}
  <p>No issues found.</p>
{{end}}

</body>
</html>
`

func formatLocation(loc *reader.Location) string {
	if loc == nil {
		return "(file-level)"
	}

	return fmt.Sprintf("L%d:%d", loc.StartLine, loc.StartColumn)
}

func add(a, b int) int {
	return a + b
}

var funcMap = template.FuncMap{
	"FormatLocation": formatLocation,
	"GetCode":        getProblemCode,
	"add":            add,
}

func (hr *HTMLReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	enrichedFindings := make([]*EnrichedFinding, len(findings))
	filePaths := make(map[string]bool)

	for i, f := range findings {
		filePaths[f.Filepath] = true
		code, err := getProblemCode(f)
		if err != nil {
			code = template.HTML(fmt.Sprintf("// Error fetching code: %v", template.HTMLEscapeString(err.Error())))
		}
		enrichedFindings[i] = &EnrichedFinding{
			Finding:      f,
			ProblemCode:  code,
			FormattedLoc: formatLocation(f.Location),
		}
	}

	totalFilesAnalyzed := len(filePaths)
	summaryItems := getSortedSummary(findings)

	reportData := HTMLReportData{
		TotalFilesAnalyzed: totalFilesAnalyzed,
		TotalFindings:      len(findings),
		Findings:           enrichedFindings,
		SummaryItems:       summaryItems,
	}

	tmpl, err := template.New("htmlReport").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %w", err)
	}

	err = tmpl.Execute(writer, reportData)
	if err != nil {
		return fmt.Errorf("error executing HTML template: %w", err)
	}
	return nil
}

type MarkdownReporter struct{}

func (md *MarkdownReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, "No issues found.")
		return err
	}

	w := bufio.NewWriter(writer)

	_, err := w.WriteString("| Severity | Rule ID | Message | File & Location |\n")
	if err != nil {
		return err
	}
	_, err = w.WriteString("|---|---|---|---|\n")
	if err != nil {
		return err
	}

	for _, finding := range findings {
		var loc string
		if finding.Location != nil {
			loc = fmt.Sprintf("`%s:%d:%d`", finding.Filepath, finding.Location.StartLine, finding.Location.StartColumn)
		} else {
			loc = fmt.Sprintf("`%s`", finding.Filepath)
		}

		message := strings.ReplaceAll(finding.Message, "|", "\\|")

		line := fmt.Sprintf("| %s | %s | %s | %s |\n",
			finding.Severity,
			finding.RuleID,
			message,
			loc)

		_, err := w.WriteString(line)
		if err != nil {
			return err
		}
	}

	summaryItems := getSortedSummary(findings)
	if len(summaryItems) > 0 {
		_, _ = w.WriteString("\n### Smell Summary\n\n")
		for _, item := range summaryItems {
			_, _ = w.WriteString(fmt.Sprintf("- **%s**: %d\n", item.RuleID, item.Count))
		}
	}

	return w.Flush()
}

func getSortedSummary(findings []*rules.Finding) []SummaryItem {

	summary := make(map[string]int)
	for _, finding := range findings {
		summary[finding.RuleID]++
	}

	var ruleIDs []string
	for ruleID := range summary {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)

	var summaryItems []SummaryItem
	for _, ruleID := range ruleIDs {
		summaryItems = append(summaryItems, SummaryItem{
			RuleID: ruleID,
			Count:  summary[ruleID],
		})
	}

	return summaryItems
}

func countSmells(findings []*rules.Finding) map[string]int {
	summary := make(map[string]int)
	for _, finding := range findings {
		summary[finding.RuleID]++
	}
	return summary
}

func countUniqueFiles(findings []*rules.Finding) int {
	files := make(map[string]struct{})
	for _, finding := range findings {
		files[finding.Filepath] = struct{}{}
	}
	return len(files)
}

func NewReporter(format ReportFormat) (Reporter, error) {
	switch format {
	case FormatJSON:
		return &JSONReporter{}, nil
	case FormatText:
		return &TextReporter{}, nil
	case FormatHTML:
		return &HTMLReporter{}, nil
	case FormatMarkdown:
		return &MarkdownReporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported report format: %s", format)
	}
}
