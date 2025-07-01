# Testing Framework for Analysis Rules

This document contains the complete documentation for the Arit project's custom testing framework for creating and running tests for Clojure code analysis rules.

## Table of Contents

1. [Overview](#overview)
2. [Framework Structure](#framework-structure)
3. [Main Components](#main-components)
4. [Quick Start Guide](#quick-start-guide)
5. [How to Implement a New Test](#how-to-implement-a-new-test)
6. [Ready-to-Use Templates](#ready-to-use-templates)
7. [How to Discover Expected Messages](#how-to-discover-expected-messages)
8. [Complete Practical Example](#complete-practical-example)
9. [Commands and Execution](#commands-and-execution)
10. [Problem Debugging](#problem-debugging)
11. [Tips and Best Practices](#tips-and-best-practices)
12. [Technical Reference](#technical-reference)

---

## Overview

The testing framework was developed to enable automated testing of code analysis rules. It isolates specific rules, runs analyses on test files, and validates whether expected problems are detected at the correct lines.

**Main features:**
- Individual rule isolation
- Message and location validation
- Integrated debugging tools
- Structured output with testify
- Ready-to-use templates

---

## Framework Structure

```
internal/test/
├── framework/           # Base framework
│   ├── framework.go    # Main structures and functions
│   └── framework_debug.go  # Debug functions
├── suite/              # Test suites by rule
│   └── *_test.go      # Go test files
└── data/               # Test data files
    └── *.clj          # Clojure files with test cases
```

---

## Main Components

### 1. Data Structures

#### `ExpectedFinding`
Defines what we expect a rule to find:
```go
type ExpectedFinding struct {
    Message   string  // Part of the expected message
    StartLine int     // Line where the problem should be detected
}
```

#### `RuleTestCase`
Defines a complete test case:
```go
type RuleTestCase struct {
    FileToAnalyze    string             // Name of the .clj file in data/
    RuleID           string             // ID of the rule to be tested
    ExpectedFindings []ExpectedFinding  // List of expected problems
}
```

### 2. Main Functions

#### `RunRuleTest(t *testing.T, tc RuleTestCase)`
Main function that runs the test:
- Isolates only the rule being tested
- Analyzes the specified file
- Compares results with expectations
- Generates individual subtests for each finding

#### `DebugRuleTest(t *testing.T, tc RuleTestCase)`
Debug function that shows all found messages:
- Runs the rule on the specified file
- Prints all found messages
- Suggests ready-to-copy code for tests
- Useful for discovering messages when you don't know them

---

## Quick Start Guide

**For developers who want to implement tests quickly!**

### Step 1: Create the Clojure data file
**File**: `internal/test/data/my_rule.clj`
```clojure
(ns my-rule-test)

;; Code that SHOULD be detected
(defn problem-here []
  (+ 1 2 3))  ; <- Line 4: problem

;; Code that should NOT be detected  
(defn ok-code []
  (map inc [1 2 3]))
```

### Step 2: Create the Go test file
**File**: `internal/test/suite/my_rule_test.go`
```go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestMyRule(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "my_rule.clj",
            RuleID:        "my-rule",  // Your rule ID
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "message text",  // Part of expected message
                    StartLine: 4,              // Line of the problem
                },
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

### Step 3: Run the test
```bash
# ALWAYS use -v for detailed output!
go test ./internal/test/suite/ -run TestMyRule -v
```

**Expected output:**
```
=== RUN   TestMyRule
=== RUN   TestMyRule/my_rule.clj
=== RUN   TestMyRule/my_rule.clj/Finding_1_line_4
--- PASS: TestMyRule (0.00s)
    --- PASS: TestMyRule/my_rule.clj (0.00s)
        --- PASS: TestMyRule/my_rule.clj/Finding_1_line_4 (0.00s)
```

### Quick Checklist
- [ ] `.clj` file created in `internal/test/data/`
- [ ] `_test.go` file created in `internal/test/suite/`
- [ ] Correct rule ID
- [ ] Lines counted correctly (start at 1)
- [ ] Message tested manually
- [ ] Test runs successfully

---

## How to Implement a New Test

### Prerequisites
Before starting, make sure that:
- [ ] The rule is already implemented in `internal/rules/`
- [ ] The rule is registered in the main analyzer
- [ ] You know the rule ID (e.g., "my-rule")
- [ ] You have code examples that the rule should detect

### Step 1: Create the Data File (.clj)

Create a file in `internal/test/data/` with code examples that your rule should detect:

```clojure
;; Example: data/my_rule.clj
(ns my-rule-test)

;; Case 1: Problem that should be detected on line 4
(defn problematic-function []
  (+ 1 2 3))  ; <- Line 4: problem here

;; Case 2: Another problem on line 8  
(defn another-function []
  (* 4 5 6))  ; <- Line 8: another problem

;; Case 3: Correct code that should NOT be detected
(defn correct-function []
  (map inc [1 2 3]))
```

### Step 2: Create the Test File (.go)

Create a file in `internal/test/suite/` following the pattern:

```go
// Example: suite/my_rule_test.go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestMyRule(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "my_rule.clj",           // File name in data/
            RuleID:        "my-rule",               // Rule ID
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "problem detected",     // Part of expected message
                    StartLine: 4,                     // Line of first problem
                },
                {
                    Message:   "another problem",     // Part of second message
                    StartLine: 8,                     // Line of second problem
                },
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

### Step 3: Run the Tests

```bash
# Run all tests
go test ./internal/test/suite/...

# Run with detailed output (RECOMMENDED)
go test ./internal/test/suite/... -v

# Run only a specific test
go test ./internal/test/suite/ -run TestMyRule -v

# Run with code coverage
go test ./internal/test/suite/... -cover -v

# Generate HTML coverage report
go test ./internal/test/suite/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

---

## Ready-to-Use Templates

### Template: Data File (.clj)

**Name**: `internal/test/data/RULE_NAME.clj`

```clojure
(ns RULE-NAME-test)

;; ===========================================
;; POSITIVE CASES (should be detected)
;; ===========================================

;; Case 1: Problem description
(defn example-problem-1 []
  ;; Code that should be detected
  (+ 1 2 3))  ; <- Line X: describe the problem here

;; Case 2: Another type of problem  
(defn example-problem-2 []
  ;; Other problematic code
  (* 4 5 6))  ; <- Line Y: another problem

;; ===========================================
;; NEGATIVE CASES (should NOT be detected)
;; ===========================================

;; Example of correct code
(defn correct-example []
  ;; This code is OK and should not generate alerts
  (map inc [1 2 3]))

;; Another correct example
(defn another-correct-example []
  (reduce + [1 2 3 4]))
```

### Template: Test File (.go)

**Name**: `internal/test/suite/RULE_NAME_test.go`

```go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestRULE_NAME(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "RULE_NAME.clj",
            RuleID:        "RULE-NAME",  // Exact rule ID
            ExpectedFindings: []framework.ExpectedFinding{
                {
                    Message:   "PART_OF_EXPECTED_MESSAGE",  // E.g., "problem detected"
                    StartLine: X,  // Line number where the problem is
                },
                {
                    Message:   "ANOTHER_PART_OF_MESSAGE",
                    StartLine: Y,  // Line of second problem
                },
                // Add more findings as needed
            },
        },
        // You can add more test cases here if needed
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

---

## How to Discover Expected Messages

### Method 1: Manual Execution (Recommended)
```bash
# Run the tool on your test file
./arit internal/test/data/your_rule.clj

# The output will show all messages, copy the most significant part
```

### Method 2: Debug Test
Use `framework.DebugRuleTest()` instead of `framework.RunRuleTest()`:

```go
func TestMyRule(t *testing.T) {
    testCase := framework.RuleTestCase{
        FileToAnalyze: "my_rule.clj",
        RuleID:        "my-rule",
        ExpectedFindings: []framework.ExpectedFinding{}, // Empty first
    }
    
    // Use DebugRuleTest instead of RunRuleTest
    framework.DebugRuleTest(t, testCase)
}
```

**Run the debug:**
```bash
go test ./internal/test/suite/ -run TestMyRule -v
```

**Expected output:**
```
--- DEBUG: Findings for rule 'my-rule' ---
Total findings: 2

Finding 1:
   Line: 4
   Message: "Parameter 'b' in function 'problem-function' is declared but never used"
   Suggested for test:
      {Message: "Parameter 'b'", StartLine: 4},

Finding 2:
   Line: 9
   Message: "Variable 'y' in let binding is declared but never used"
   Suggested for test:
      {Message: "Variable 'y'", StartLine: 9},
--- END DEBUG ---
```

**Perfect!** The debug already suggests ready-to-copy code!

### Method 3: Empty Test
Leave `ExpectedFindings: []` empty and run the test to see how many findings exist.

### Tips for Messages

#### 1. Very long messages?
If the message is too long, use only the most significant part:

```
Real message: "Parameter 'b' in function 'problem-function' is declared but never used. Consider removing it or using it."

Use only: "Parameter 'b'"
```

#### 2. Multiple variations?
If your rule generates slightly different messages, choose the common part:

```
"Parameter 'a' is unused"
"Parameter 'b' is unused"  
"Parameter 'x' is unused"

Use: "is unused"
```

#### 3. Case-sensitive!
Remember that comparison is case-sensitive:

```go
// Will fail
{Message: "PARAMETER 'b'", StartLine: 4}

// Will pass  
{Message: "Parameter 'b'", StartLine: 4}
```

---

## Complete Practical Example

### Scenario: New Rule "unused-variable"

Imagine you implemented a rule that detects unused variables, but you don't know exactly what messages it generates.

### Step 1: Create the data file

```clojure
;; internal/test/data/unused_variable.clj
(ns unused-variable-test)

;; Case with unused variable
(defn problem-function [a b]  ; b is not used
  (* a 2))

;; Case with let and unused variable  
(defn another-function []
  (let [x 10
        y 20]  ; y is not used
    x))
```

### Step 2: Manual Execution

```bash
$ ./arit internal/test/data/unused_variable.clj
[WARN] unused-variable: Parameter 'b' in function 'problem-function' is declared but never used [unused_variable.clj:4:1]
[WARN] unused-variable: Variable 'y' in let binding is declared but never used [unused_variable.clj:9:8]
```

**Result**: Now you know the messages!

### Step 3: Implement the final test

```go
package suite

import (
    "testing"
    "github.com/thlaurentino/arit/internal/test/framework"
)

func TestUnusedVariable(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "unused_variable.clj",
            RuleID:        "unused-variable",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "Parameter 'b'", StartLine: 4},
                {Message: "Variable 'y'", StartLine: 9},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

---

## Commands and Execution

### Basic Commands
```bash
# Run only your test
go test ./internal/test/suite/ -run TestRULE_NAME

# Run with detailed output
go test ./internal/test/suite/ -run TestRULE_NAME -v

# Run all tests
go test ./internal/test/suite/...

# Test your rule manually
./arit internal/test/data/RULE_NAME.clj
```

### Advanced Commands
```bash
# Run with coverage
go test ./internal/test/suite/... -cover -v

# Generate HTML coverage report
go test ./internal/test/suite/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run with timeout
go test ./internal/test/suite/... -v -timeout=30s

# Run in parallel
go test ./internal/test/suite/... -v -parallel=4
```

### Test Output

The framework uses the testify library to generate structured output. When run with `-v`, you'll see:

```
=== RUN   TestExplicitRecursion
=== RUN   TestExplicitRecursion/explicit_recursion.clj
=== RUN   TestExplicitRecursion/explicit_recursion.clj/Finding_1_line_6
=== RUN   TestExplicitRecursion/explicit_recursion.clj/Finding_2_line_12
--- PASS: TestExplicitRecursion (0.00s)
    --- PASS: TestExplicitRecursion/explicit_recursion.clj (0.00s)
        --- PASS: TestExplicitRecursion/explicit_recursion.clj/Finding_1_line_6 (0.00s)
        --- PASS: TestExplicitRecursion/explicit_recursion.clj/Finding_2_line_12 (0.00s)
```

Each finding is tested individually as a subtest, allowing you to identify exactly which validation failed.

---

## Problem Debugging

### Test Fails: "Expected finding on line X, but none was found"
**Possible causes:**
- Incorrect line in test
- Rule doesn't detect expected problem
- Incorrect data file

**Solutions:**
- Check if the line is correct (editors show line numbers)
- Run manually: `./arit internal/test/data/your_file.clj`
- Use `DebugRuleTest` to see all findings

### Test Fails: "Finding message does not contain expected text"
**Possible causes:**
- Expected message doesn't match the real one
- Case-sensitive
- Message too specific

**Solutions:**
- Run manually to see the real message
- Use only part of the message, not the complete message
- Check case-sensitivity

### Test Fails: "Incorrect number of findings"
**Possible causes:**
- Negative cases being detected erroneously
- Positive cases not being detected
- Incorrect count

**Solutions:**
- Use `DebugRuleTest` to see how many findings exist
- Check if negative cases are being detected erroneously
- Confirm all positive cases are being detected

### Recommended Debug Flow

1. **Create the .clj file** with test cases
2. **Run manually** `./arit file.clj`
3. **If it doesn't work**, use `DebugRuleTest()`
4. **Copy the suggestions** generated by debug
5. **Replace with `RunRuleTest()`** in the final test
6. **Run the test** to confirm it passes

### Common Problems

| Error | Solution |
|-------|----------|
| "Expected finding on line X, but none was found" | Check if the line is correct |
| "Finding message does not contain expected text" | Run manually and copy part of the real message |
| "Incorrect number of findings" | Count how many problems your rule actually detects |

---

## Tips and Best Practices

### **DOs**

1. **Descriptive names**: Use clear names for files and functions
2. **Explanatory comments**: Document each test case in the .clj file
3. **Varied cases**: Include positive cases (should be detected) and negative ones (shouldn't)
4. **Specific messages**: Use specific parts of the message in `ExpectedFinding.Message`
5. **Correct lines**: Verify that `StartLine` corresponds to the real problem line
6. **Test first**: Use discovery strategies before writing tests

### **DON'Ts**

1. **Don't use generic messages**: Avoid very vague messages like "error"
2. **Don't ignore negative cases**: Always test that correct code is not detected
3. **Don't hardcode paths**: Use only relative file names
4. **Don't test multiple rules**: Each test should focus on one specific rule
5. **Don't forget to run**: Always run tests before committing

### Implementation Tips

#### 1. How to discover the exact rule message?
```bash
# Run the tool on your test file
./arit internal/test/data/your_rule.clj

# The output will show the real message, use part of it in the test
```

#### 2. How to count lines correctly?
- Use an editor that shows line numbers
- Remember: the first line is 1, not 0
- Count the line where the PROBLEM is, not where the function starts

#### 3. How to test cases without problems?
```go
{
    FileToAnalyze: "clean_code.clj",
    RuleID:        "your-rule",
    ExpectedFindings: []framework.ExpectedFinding{}, // Empty list!
}
```

#### 4. How to debug failing tests?

1. **Run manually first**:
   ```bash
   ./arit internal/test/data/your_file.clj
   ```

2. **Compare real output with expected**:
    - Message: contains expected text?
    - Line: is it on the correct line?

3. **Run the test with -v**:
   ```bash
   go test ./internal/test/suite/ -run YourTest -v
   ```

4. **Use DebugRuleTest**:
   ```go
   // Temporarily replace RunRuleTest with DebugRuleTest
   framework.DebugRuleTest(t, tc)
   ```

### Common Errors

1. **Wrong file name**: Make sure the name in `FileToAnalyze` matches the real file
2. **Wrong rule ID**: Use exactly the same ID defined in the rule implementation
3. **Wrong line**: Count lines correctly (editors help!)
4. **Message too specific**: Use only part of the message, not the complete message
5. **Forgetting negative cases**: Always test that correct code doesn't generate alerts

---

## Technical Reference

### Advanced Features

#### Multiple Test Cases
```go
func TestMyRule(t *testing.T) {
    testCases := []framework.RuleTestCase{
        {
            FileToAnalyze: "simple_case.clj",
            RuleID:        "my-rule",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "simple problem", StartLine: 3},
            },
        },
        {
            FileToAnalyze: "complex_case.clj",
            RuleID:        "my-rule",
            ExpectedFindings: []framework.ExpectedFinding{
                {Message: "complex problem", StartLine: 5},
                {Message: "another problem", StartLine: 10},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.FileToAnalyze, func(t *testing.T) {
            framework.RunRuleTest(t, tc)
        })
    }
}
```

#### Testing Negative Cases (No Problems)
```go
{
    FileToAnalyze: "clean_code.clj",
    RuleID:        "my-rule",
    ExpectedFindings: []framework.ExpectedFinding{}, // Empty list
}
```

### Validation Structure

The framework validates:

1. **Correct number of findings**: Should match the number of `ExpectedFindings`
2. **Correct line**: Each finding should be on the specified line
3. **Correct message**: Should contain the text specified in `Message`
4. **Correct RuleID**: Should match the test's `RuleID`

### Debug Checklist

- [ ] .clj file created with representative cases
- [ ] Manual execution tested
- [ ] Debug test executed (if needed)
- [ ] Messages copied correctly
- [ ] Lines verified (start at 1)
- [ ] Final test executed successfully
- [ ] `DebugRuleTest` removed/commented

---

## Conclusion

This framework provides a structured and reliable way to test analysis rules. Use the discovery strategies to find messages, implement varied cases, and always run tests before committing.

**Remember**:
- Start simple and incrementally improve
- It's better to have few well-tested cases than many poorly implemented ones
- Debug is a temporary tool - use it to discover messages and then return to `RunRuleTest()`
- Test both positive and negative cases

For questions or problems, check the example files in `internal/test/suite/` or use the available debug functions. 