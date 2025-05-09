package reporter

import (
	"encoding/json"
	"fmt"
	"html/template"
	io "io"
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
	return nil
}

type HTMLReporter struct{}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
<title>Arit Analysis Report</title>
<style>
  body { font-family: sans-serif; }
  table { border-collapse: collapse; width: 100%; margin-top: 1em; }
  th, td { border: 1px solid #ddd; padding: 10px 8px; text-align: left; }
  th { background-color: #f2f2f2; font-weight: bold; text-align: center; }
  tbody tr:nth-child(even) { background-color: #f9f9f9; }
  tbody tr:hover { background-color: #f1f1f1; }
  .severity-ERROR { color: red; font-weight: bold; }
  .severity-WARNING { color: orange; }
  .severity-INFO { color: blue; }
  .location { font-family: monospace; }
</style>
</head>
<body>

<h1>Arit Analysis Report</h1>

{{if .}}
  <table>
    <thead>
      <tr>
        <th>Severity</th>
        <th>Rule ID</th>
        <th>Message</th>
        <th>File</th>
        <th>Location</th>
      </tr>
    </thead>
    <tbody>
      {{range .}}
      <tr>
        <td class="severity-{{.Severity}}">{{.Severity}}</td>
        <td>{{.RuleID}}</td>
        <td>{{.Message}}</td>
        <td>{{.Filepath}}</td>
        <td class="location">{{FormatLocation .Location}}</td>
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

var funcMap = template.FuncMap{
	"FormatLocation": formatLocation,
}

func (hr *HTMLReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %w", err)
	}

	err = tmpl.Execute(writer, findings)
	if err != nil {
		return fmt.Errorf("error executing HTML template: %w", err)
	}
	return nil
}

type MarkdownReporter struct{}

func (md *MarkdownReporter) Report(findings []*rules.Finding, writer io.Writer) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, "# Arit Analysis Report\n\nNo issues found.")
		return err
	}

	_, err := fmt.Fprintln(writer, "# Arit Analysis Report\n")
	if err != nil {
		return fmt.Errorf("error writing markdown header: %w", err)
	}
	_, err = fmt.Fprintln(writer, "| Severity | Rule ID | Message | File | Location |")
	if err != nil {
		return fmt.Errorf("error writing markdown table header: %w", err)
	}
	_, err = fmt.Fprintln(writer, "|---|---|---|---|---|")
	if err != nil {
		return fmt.Errorf("error writing markdown table separator: %w", err)
	}

	for _, finding := range findings {
		loc := "(file-level)"
		if finding.Location != nil {
			loc = fmt.Sprintf("`%s:%d:%d`", finding.Filepath, finding.Location.StartLine, finding.Location.StartColumn)
		} else {
			loc = fmt.Sprintf("`%s`", finding.Filepath)
		}

		message := strings.ReplaceAll(finding.Message, "|", "\\|")

		line := fmt.Sprintf("| %s | %s | %s | `%s` | %s |",
			finding.Severity,
			finding.RuleID,
			message,
			finding.Filepath,

			loc)

		filepathStr := fmt.Sprintf("`%s`", finding.Filepath)
		locationStr := "(file-level)"
		if finding.Location != nil {
			locationStr = fmt.Sprintf("`L%d:%d`", finding.Location.StartLine, finding.Location.StartColumn)
		}

		line = fmt.Sprintf("| %s | %s | %s | %s | %s |",
			finding.Severity,
			finding.RuleID,
			message,
			filepathStr,
			locationStr)

		_, err = fmt.Fprintln(writer, line)
		if err != nil {
			return fmt.Errorf("error writing markdown finding row: %w", err)
		}
	}

	return nil
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
