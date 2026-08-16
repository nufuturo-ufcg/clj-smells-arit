# ARIT - Static Code Analyzer for Clojure

```
###############
    • 
┏┓┏┓┓╋
┗┻┛ ┗┗
      
###############
```

ARIT is a static code analyzer for Clojure that detects code smells, anti-patterns, and quality issues. Built in Go, ARIT employs a hybrid semantic engine designed to achieve high precision by mitigating structural false positives common in purely AST-based static analysis.

## Features

- **Semantic Engine Architecture**: Mitigates false positives via reader emulation (AST pruning of comments and discards), lexical scope tracking, and control-flow graphing for macros.
- **Interoperable Inline Suppression**: Supports standard suppression directives (`clj-kondo/ignore`, `nosonar`, `eslint-disable`, and `arit:disable-next-line`) to manage intentional patterns.
- **42+ Analysis Rules**: Comprehensive detection covering functional paradigms, Clojure idioms, and performance anti-patterns.
- **Parallel Analysis**: Concurrent file processing for large codebases.
- **Experimental Semantic Flags**: Support for macro micro-expansion and foundational flags for future cross-namespace and data-flow analyses.

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

Analyze an entire directory (recursive):
```bash
./arit src/
```

### Output Formats

ARIT supports multiple output formats:

#### Text Output (Default)
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

### List Available Rules

View all available analysis rules:
```bash
./arit list-rules
```

## Configuration

ARIT uses an optional `.arit.yaml` configuration file to customize analysis behavior. The tool automatically searches for this file starting from the analyzed directory and moving up the directory hierarchy.

### Sample Configuration

```yaml
enabled-rules:
  long-function: true
  long-parameter-list: true
  duplicated-code-global: false

rule-config:
  long-function:
    max-lines: 20
    count-let-bindings: true
```

## Architecture

ARIT is built with a modular architecture:

```
├── cmd/                    # CLI interface (Cobra)
├── internal/
│   ├── analyzer/          # Core analysis engine and semantic passes
│   ├── config/            # Configuration management
│   ├── reader/            # Clojure parser integration (goclj)
│   ├── reporter/          # Output formatting
│   └── rules/             # Analysis rules implementation
└── main.go                # Application entry point
```

## Contributing

1. Create a new rule file in `internal/rules/`
2. Implement the `CheckerRule` interface
3. Register the rule in the init function
4. Add tests and examples

## Dependencies

- [goclj](https://github.com/cespare/goclj) - Clojure parser for Go
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML v3](https://gopkg.in/yaml.v3) - Configuration file parsing
