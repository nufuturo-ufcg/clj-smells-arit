# ARIT - Static Code Analyzer for Clojure

```
###############
    * 
┏┓┏┓┓╋
┗┻┛ ┗┗
       
###############
```

ARIT is a comprehensive static code analyzer for Clojure that detects code smells, anti-patterns, and quality issues in your codebase. Built in Go for performance and reliability, ARIT helps maintain clean, idiomatic Clojure code by identifying potential problems before they impact your application.

## Features

- **42+ Analysis Rules**: Comprehensive detection of code smells, anti-patterns, and quality issues
- **Multiple Output Formats**: Text, JSON, HTML, Markdown, CSV, and Summary support
- **Parallel Analysis**: High-performance concurrent file processing
- **Configurable Rules**: Fine-tune analysis with YAML configuration
- **Rich Context**: Detailed location information and code snippets in reports
- **Clojure-Specific**: Tailored for functional programming patterns and Clojure idioms

## Installation

### Prerequisites

- Go 1.21+ (for building from source)
- Clojure files (.clj, .cljs, .cljc) to analyze

### Building from Source

```bash
# Clone the repository
git clone https://github.com/thlaurentino/arit.git
cd arit

# Build the binary
go build -o arit .

# Verify installation
./arit --help
```

## Usage

### Basic Analysis

Analyze a single file:

```bash
./arit path/to/your/file.clj
```

Analyze multiple files:

```bash
./arit file1.clj file2.clj file3.clj
```

Analyze an entire directory (recursive):

```bash
./arit src/
```

### Output Formats

ARIT supports multiple output formats for different use cases:

| Format | Flag Value | Description |
|--------|------------|-------------|
| Summary | `summary` | Aggregated count by rule (default) |
| Text | `text` | Human-readable list of findings |
| JSON | `json` | Machine-readable JSON for CI/CD |
| HTML | `html` | Interactive HTML report |
| Markdown | `markdown` | Markdown table format |
| CSV | `csv` | CSV summary of findings |

#### Text Output

```bash
./arit --format text src/
```

#### JSON Output

```bash
./arit --format json src/ > analysis-results.json
```

#### HTML Report

```bash
./arit --format html src/ > report.html
```

#### Markdown Report

```bash
./arit --format markdown src/ > ANALYSIS.md
```

### Global Flags

| Flag | Shortcut | Description |
|------|----------|-------------|
| `--format` | `-f` | Output format (default: summary) |
| `--verbose` | `-v` | Enable verbose output |
| `--timing` | `-t` | Show execution time |
| `--quiet` | `-q` | Suppress banner and progress output |

## Commands Reference

### Main Analysis

```bash
./arit <file-or-directory> [files-or-directories...]
```

This is the primary command that analyzes Clojure files for code smells and issues.

### List Rules

```bash
./arit list-rules
```

Lists all available analysis rules with their names in a simplified format.

### Info Rules

```bash
./arit info-rules
```

Lists all available analysis rules with detailed information including:
- Rule ID
- Name
- Severity level
- Full description

### Statistics

```bash
./arit stats <path>
```

Collects statistics from a Clojure project, including:
- Lines of code per function
- Parameter count per function
- Maximum nesting depth
- Message chain length
- Consecutive primitive parameters

Outputs aggregated CSV files for each metric.

Flags:
- `--output-dir, -o`: Directory to save output CSV files (default: new directory with timestamp)
- `--raw`: Output raw, non-aggregated stats to a single CSV file

### Generate Configuration

```bash
./arit genconf
```

Generates a default `.arit.yaml` configuration file with all available rules and their default settings.

## Configuration

ARIT uses an optional `.arit.yaml` configuration file to customize analysis behavior. The tool automatically searches for this file starting from the analyzed directory and moving up the directory hierarchy.

### Sample Configuration

Create a `.arit.yaml` file in your project root:

```yaml
# Arit configuration file.
#
# You can enable or disable rules by setting them to true or false in the 'enabled_rules' section.
# Specific parameters for rules can be configured in the 'rule_config' section.

enabled-rules:
  long-function: true
  long-parameter-list: true
  duplicated-code-global: false
  shotgun-surgery: true

rule-config:
  long-function:
    max-lines: 20
    count-let-bindings: true

  long-parameter-list:
    max-parameters: 4

  data-clumps:
    min-occurrences: 3
    min-parameters: 3

  nested-forms:
    max-depth: 4
```

### Configuration Structure

| Section | Type | Description |
|---------|------|-------------|
| `enabled-rules` | map[string]bool | Enable or disable specific rules by ID |
| `rule-config` | map[string]map[string]interface{} | Configure rule-specific parameters |

## Architecture

ARIT is built with a modular architecture:

```
├── cmd/                    # CLI interface (Cobra)
│   ├── root.go            # Main entry point, parallel processing
│   ├── list.go            # list-rules command
│   ├── info.go            # info-rules command
│   ├── stats.go           # stats command
│   └── genconf.go        # genconf command
├── internal/
│   ├── analyzer/          # Core analysis engine
│   │   ├── analyzer.go   # Parsing, scope, symbol resolution, rule execution
│   │   └── scope.go     # Scope management and symbol lookup
│   ├── config/           # Configuration management
│   │   └── config.go    # YAML config loading
│   ├── reader/           # Clojure parser integration
│   │   ├── parser.go    # File parsing wrapper
│   │   └── ast.go      # RichNode definitions
│   ├── reporter/         # Output formatting
│   │   └── reporter.go # Multiple output format implementations
│   └── rules/            # Analysis rules
│       ├── rules.go     # Rule registry and interfaces
│       ├── types.go     # Finding and Severity types
│       └── [45 rules]   # Individual rule implementations
└── main.go                # Application entry point
```

### Data Representation

ARIT uses a rich Abstract Syntax Tree (AST) to represent Clojure code:

#### RichNode

The core AST node type representing elements in Clojure code:

```go
type RichNode struct {
    Type     NodeType    // NodeType: List, Vector, Map, Set, Symbol, Keyword, etc.
    Value    string      // String value for leaf nodes (symbols, keywords, strings)
    Location *Location   // Source location (StartLine, StartColumn, EndLine, EndColumn)
    Children []*RichNode // Child nodes
    Metadata *RichNode   // Metadata attached to node
    ResolvedDefinition *RichNode // Linked definition for resolved symbols
    SymbolRef interface{}       // Symbol reference info (SymbolInfo or NamespaceAlias)
}
```

#### Location

Source code location information:

```go
type Location struct {
    StartLine   int // 1-indexed line number
    StartColumn int // 1-indexed column number
    EndLine     int // 1-indexed line number
    EndColumn   int // 1-indexed column number
}
```

#### Scope

Scope represents a lexical context for symbol definitions:

```go
type Scope struct {
    parent          *Scope
    symbols         map[string]*SymbolInfo   // Local symbol definitions
    aliases         map[string]*NamespaceAlias // Namespace aliases (require :as)
    referredSymbols map[string]*ReferredSymbol // Referred symbols (require :refer)
}
```

#### SymbolInfo

Information about a defined symbol:

```go
type SymbolInfo struct {
    Name            string     // Symbol name
    Definition      *RichNode // AST node where symbol is defined
    Type            SymbolType // function, variable, parameter, namespace, etc.
    IsPrivate       bool       // Whether symbol is private (defn-)
    IsUsed          bool       // Whether symbol is referenced
    OriginNamespace string     // Original namespace for referred symbols
}
```

#### Finding

A finding represents a rule violation:

```go
type Finding struct {
    RuleID   string           // Unique rule identifier
    Message  string           // Human-readable message
    Filepath string           // File where issue was found
    Location *Location        // Source location
    Severity Severity         // WARNING, INFO, or HINT
}
```

### Analysis Workflow

1. **Parsing**: ARIT uses the goclj parser to parse Clojure source files into an initial AST
2. **Rich Tree Building**: The parser output is transformed into RichNode structures with location and metadata
3. **Namespace Extraction**: The ns form is parsed to extract namespace name, aliases, and referred symbols
4. **Scope Building**: A global scope is created with namespace aliases and referred symbols
5. **Definition Collection**: Function definitions (defn, defn-, fn), variable definitions (def, defonce), and let/loop bindings are collected into the scope
6. **Symbol Resolution**: Symbol references are resolved to their definitions using the scope chain
7. **Rule Execution**: Each enabled rule is applied to every node in the AST. Rules receive a context map with:
   - `scope`: Current scope for symbol lookup
   - `isInsideFunction`: Whether traversal is inside a function body
   - `isInsideLet`: Whether traversal is inside a let binding
   - `isInsideLoop`: Whether traversal is inside a loop
   - `isInsideBinding`: Whether traversal is inside a binding form
   - `isInsideDosync`: Whether traversal is inside a dosync block
   - `isInEagerContext`: Whether traversal is inside an eager context (doall, dorun, etc.)
   - `parent`: Parent node in AST
8. **Report Generation**: Findings are sorted by file and location, then formatted according to the selected output format

### Context for Rules

When rules are executed, they receive a context map containing:

| Key | Type | Description |
|-----|------|-------------|
| `scope` | *Scope | Current scope for symbol resolution |
| `isInsideFunction` | bool | True if node is inside a function body |
| `isInsideLet` | bool | True if node is inside a let binding |
| `isInsideLoop` | bool | True if node is inside a loop form |
| `isInsideBinding` | bool | True if node is inside a binding form |
| `isInsideDosync` | bool | True if node is inside a dosync block |
| `isInEagerContext` | bool | True if inside doall, dorun, or other eager consumer |
| `parent` | *RichNode | Parent node in the AST |

## Rule Categories

ARIT's rules are organized into several categories:

### Traditional Code Smells

- Long Function
- Long Parameter List
- Data Clumps
- Duplicated Code
- Feature Envy
- Message Chains
- Middle Man
- Shotgun Surgery
- Divergent Change

### Functional Programming Specific

- Explicit Recursion
- Lazy Side Effects
- Hidden Side Effects
- Immutability Violation
- Thread Ignorance
- Trivial Lambda

### Clojure-Specific

- Namespaced Keys Neglect
- Direct Use of clojure.lang.RT
- Production doall Usage
- Unnecessary Into
- Verbose Checks
- Improper Emptiness Check

### Performance and Efficiency

- Inappropriate Collection Usage
- Inefficient Filtering
- Linear Collection Scan
- Potentially Inefficient Generator

### Code Quality and Style

- Comment Quality Analysis
- Redundant Do Block
- Conditional Build-Up
- Nested Forms
- Primitive Obsession

## Example Output

### Text Format

```
[WARNING] long-function: Function 'process-data' has 25 lines, exceeding the limit of 20 [src/core.clj:15:1]
[INFO] namespaced-keys-neglect: Non-namespaced keyword ':name' detected. Consider using :myapp/name [src/core.clj:23:15]
[HINT] thread-ignorance: Nested function calls detected that could benefit from threading macro (->) [src/core.clj:45:8]

---
Smell Summary:
- long-function: 1
- namespaced-keys-neglect: 1
- thread-ignorance: 1
```

### JSON Format

```json
[
  {
    "rule_id": "long-function",
    "message": "Function 'process-data' has 25 lines, exceeding the limit of 20",
    "filepath": "src/core.clj",
    "location": {
      "start_line": 15,
      "start_col": 1,
      "end_line": 40,
      "end_col": 2
    },
    "severity": "WARNING"
  }
]
```

### HTML Report

The HTML report provides an interactive table with:
- Severity color coding
- Rule ID and message
- File location with line/column
- Code snippets with syntax highlighting
- Summary statistics

## Creating Rules

To add a new analysis rule to ARIT, create a new file in the `internal/rules/` directory.

### Rule Structure

A rule must implement the `CheckerRule` interface:

```go
type CheckerRule interface {
    Meta() Rule
    Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding
}
```

### Example: Creating a New Rule

```go
package rules

import (
    "fmt"
    "github.com/thlaurentino/arit/internal/reader"
)

type MyCustomRule struct {
    Rule
    MaxThreshold int `yaml:"max_threshold"`
}

func (r *MyCustomRule) Meta() Rule {
    return r.Rule
}

func (r *MyCustomRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
    // Check if node matches the pattern we're looking for
    if node.Type != reader.NodeList || len(node.Children) == 0 {
        return nil
    }

    // Check if it's a defn form
    if node.Children[0].Value == "defn" && len(node.Children) > 1 {
        funcName := node.Children[1].Value
        
        // Access context information
        if isInsideFunc, ok := context["isInsideFunction"].(bool); isInsideFunc {
            // Rule implementation logic here
        }

        // Access scope for symbol resolution
        if scope, ok := context["scope"].(*analyzer.Scope); ok {
            // Look up symbols in scope
        }

        // Return finding if issue detected
        return &Finding{
            RuleID:   r.ID,
            Message:  fmt.Sprintf("Custom rule triggered for function %s", funcName),
            Filepath: filepath,
            Location: node.Location,
            Severity: r.Severity,
        }
    }

    return nil
}

func init() {
    // Register the rule
    RegisterRule(&MyCustomRule{
        Rule: Rule{
            ID:          "my-custom-rule",
            Name:        "My Custom Rule",
            Description: "Description of what this rule detects",
            Severity:    SeverityWarning,
        },
        MaxThreshold: 10,
    })
}
```

### Rule Metadata

The `Rule` struct defines the rule's metadata:

```go
type Rule struct {
    ID          string   // Unique identifier (e.g., "long-function")
    Name        string   // Human-readable name (e.g., "Long Function")
    Description string   // Detailed description of what the rule detects
    Severity    Severity // WARNING, INFO, or HINT
}
```

### Accessing Node Information

Within the `Check` method, you can access:

- **Node Type**: `node.Type` - Compare against `reader.NodeList`, `reader.NodeSymbol`, etc.
- **Node Value**: `node.Value` - For symbols, keywords, strings
- **Node Location**: `node.Location.StartLine`, `node.Location.StartColumn`, etc.
- **Children**: `node.Children` - List of child RichNode pointers
- **Resolved Definition**: `node.ResolvedDefinition` - For resolved symbols
- **Symbol Reference**: `node.SymbolRef` - Can be cast to `*SymbolInfo` or `*NamespaceAlias`

### Rule Severity Levels

| Level | Description |
|-------|-------------|
| WARNING | Critical issues that likely cause runtime problems |
| INFO | Important issues that should be addressed |
| HINT | Minor suggestions for better idioms |

### Configuration Support

To add configurable parameters to your rule:

1. Add fields with YAML tags to your rule struct:

```go
type MyRule struct {
    Rule
    MaxLines int `yaml:"max_lines"`
    MinParams int `yaml:"min_parameters"`
}
```

2. Use `config.GetRuleSettingInt` or `config.GetRuleSettingBool` in the analyzer to pass configured values to your rule.

## Dependencies

- [goclj](https://github.com/cespare/goclj) - Clojure parser for Go
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML v3](https://gopkg.in/yaml.v3) - Configuration file parsing

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [goclj](https://github.com/cespare/goclj) for robust Clojure parsing
- Inspired by various code smell catalogs and static analysis tools
- Based on functional programming best practices and Clojure idioms
