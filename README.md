# ARIT - Static Code Analyzer for Clojure

```
###############
    • 
┏┓┏┓┓╋
┗┻┛ ┗┗
      
###############
```

**ARIT** is a comprehensive static code analyzer for Clojure that detects code smells, anti-patterns, and quality issues in your codebase. Built in Go for performance and reliability, ARIT helps maintain clean, idiomatic Clojure code by identifying potential problems before they impact your application.

## 🚀 Features

- **42+ Analysis Rules**: Comprehensive detection of code smells, anti-patterns, and quality issues
- **Multiple Output Formats**: Text, JSON, HTML, and Markdown support
- **Parallel Analysis**: High-performance concurrent file processing
- **Configurable Rules**: Fine-tune analysis with YAML configuration
- **Rich Context**: Detailed location information and code snippets in reports
- **Clojure-Specific**: Tailored for functional programming patterns and Clojure idioms

## 📦 Installation

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

### Using Pre-built Binary

If available, download the pre-built binary for your platform from the releases page.

## 🔧 Usage

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

#### Text Output (Default)
```bash
./arit --format text src/
```

#### JSON Output (for CI/CD integration)
```bash
./arit --format json src/ > analysis-results.json
```

#### HTML Report (for detailed review)
```bash
./arit --format html src/ > report.html
```

#### Markdown Report (for documentation)
```bash
./arit --format markdown src/ > ANALYSIS.md
```

### List Available Rules

View all available analysis rules:
```bash
./arit list-rules
```

## ⚙️ Configuration

ARIT uses a `.arit.yaml` configuration file to customize analysis behavior. The tool automatically searches for this file starting from the analyzed directory and moving up the directory hierarchy.

### Sample Configuration

Create a `.arit.yaml` file in your project root:

```yaml
# Enable/disable specific rules
enabled-rules:
  long-function: true
  long-parameter-list: true
  duplicated-code-global: false
  shotgun-surgery: true

# Configure rule-specific settings
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

### Rule Categories

ARIT's rules are organized into several categories:

#### **Traditional Code Smells**
- Long Function
- Long Parameter List
- Data Clumps
- Duplicated Code
- Feature Envy
- Message Chains
- Middle Man
- Shotgun Surgery
- Divergent Change

#### **Functional Programming Specific**
- Explicit Recursion
- Lazy Side Effects
- Hidden Side Effects
- Immutability Violation
- Thread Ignorance
- Trivial Lambda

#### **Clojure-Specific**
- Namespaced Keys Neglect
- Direct Use of clojure.lang.RT
- Production doall Usage
- Unnecessary Into
- Verbose Checks
- Improper Emptiness Check

#### **Performance & Efficiency**
- Inappropriate Collection Usage
- Inefficient Filtering
- Linear Collection Scan
- Potentially Inefficient Generator

#### **Code Quality & Style**
- Comment Quality Analysis
- Redundant Do Block
- Conditional Build-Up
- Nested Forms
- Primitive Obsession

## 📊 Example Output

### Text Format
```
[WARNING] long-function: Function 'process-data' has 25 lines, exceeding the limit of 20 [src/core.clj:15:1]
[INFO] namespaced-keys-neglect: Non-namespaced keyword ':name' detected. Consider using :myapp/name [src/core.clj:23:15]
[HINT] thread-ignorance: Nested function calls detected that could benefit from threading macro (->) [src/core.clj:45:8]
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

## 🏗️ Architecture

ARIT is built with a modular architecture:

```
├── cmd/                    # CLI interface (Cobra)
├── internal/
│   ├── analyzer/          # Core analysis engine
│   ├── config/            # Configuration management
│   ├── reader/            # Clojure parser integration
│   ├── reporter/          # Output formatting
│   └── rules/             # Analysis rules implementation
└── main.go                # Application entry point
```

### Key Components

- **Parser**: Uses [goclj](https://github.com/cespare/goclj) for robust Clojure code parsing
- **Analyzer**: Builds rich AST with scope analysis and symbol resolution
- **Rules Engine**: Pluggable rule system with configurable parameters
- **Reporter**: Multiple output formats with detailed context

## 🔍 Analysis Rules

### Rule Severity Levels

- **ERROR**: Critical issues that likely cause runtime problems
- **WARNING**: Important issues that should be addressed
- **INFO**: Informational suggestions for code improvement
- **HINT**: Minor suggestions for better idioms

### Sample Rules

#### Long Function Detection
```clojure
;; This function would trigger the long-function rule
(defn process-users [users]
  (let [filtered (filter #(> (:age %) 18) users)
        formatted (map #(str (:first-name %) " " (:last-name %)) filtered)
        report (map #(hash-map :full-name %1 :age (:age %2)) formatted filtered)]
    ;; ... many more lines
    report))
```

#### Namespaced Keys Neglect
```clojure
;; Problematic: non-namespaced keys
{:name "John" :email "john@example.com"}

;; Better: namespaced keys
{:user/name "John" :user/email "john@example.com"}
```

#### Thread Ignorance
```clojure
;; Problematic: nested function calls
(filter even? (map inc (range 10)))

;; Better: using threading macro
(->> (range 10)
     (map inc)
     (filter even?))
```

## 🤝 Contributing

### Adding New Rules

1. Create a new rule file in `internal/rules/`
2. Implement the `CheckerRule` interface
3. Register the rule in the init function
4. Add tests and examples

Example rule structure:
```go
type MyRule struct {
    rule rules.Rule
}

func (r *MyRule) Meta() rules.Rule {
    return r.rule
}

func (r *MyRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
    // Implementation here
    return nil
}

func init() {
    rules.RegisterRule(&MyRule{
        rule: rules.Rule{
            ID:          "my-rule",
            Name:        "My Custom Rule",
            Description: "Detects my specific pattern",
            Severity:    rules.SeverityWarning,
        },
    })
}
```

### Development Setup

```bash
# Install dependencies
go mod download

# Build the project
go build -o arit .




## 🔗 Dependencies

- [goclj](https://github.com/cespare/goclj) - Clojure parser for Go
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML v3](https://gopkg.in/yaml.v3) - Configuration file parsing

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built with [goclj](https://github.com/cespare/goclj) for robust Clojure parsing
- Inspired by various code smell catalogs and static analysis tools
- Based on functional programming best practices and Clojure idioms

---

