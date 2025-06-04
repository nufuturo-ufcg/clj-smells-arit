package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type LibraryLockerRule struct {
	Rule
	ExcludedLibraries []string `json:"excluded_libraries" yaml:"excluded_libraries"`
	MinParamCount     int      `json:"min_param_count" yaml:"min_param_count"`
}

func (r *LibraryLockerRule) Meta() Rule {
	return r.Rule
}

func (r *LibraryLockerRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || (firstChild.Value != "defn" && firstChild.Value != "defn-") {
		return nil
	}

	funcNameNode := node.Children[1]
	if funcNameNode.Type != reader.NodeSymbol {
		return nil
	}
	funcName := funcNameNode.Value

	var paramsNode *reader.RichNode
	var bodyStartIndex int

	for i := 2; i < len(node.Children); i++ {
		child := node.Children[i]
		if child.Type == reader.NodeVector {
			paramsNode = child
			bodyStartIndex = i + 1
			break
		}
	}

	if paramsNode == nil || bodyStartIndex >= len(node.Children) {
		return nil
	}

	params := paramsNode.Children

	if len(params) < r.MinParamCount {
		return nil
	}

	bodyNodes := node.Children[bodyStartIndex:]

	var significantBodyNode *reader.RichNode
	for _, bodyNode := range bodyNodes {
		if bodyNode.Type != reader.NodeComment && bodyNode.Type != reader.NodeNewline {
			if significantBodyNode != nil {

				return nil
			}
			significantBodyNode = bodyNode
		}
	}

	if significantBodyNode == nil {
		return nil
	}

	libraryCall := r.extractLibraryCall(significantBodyNode)
	if libraryCall == nil {
		return nil
	}

	if r.isLibraryLocker(params, libraryCall, funcName) {
		return &Finding{
			RuleID:   r.ID,
			Message:  r.formatMessage(funcName, libraryCall),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

type LibraryCall struct {
	Library    string
	Function   string
	Arguments  []*reader.RichNode
	FullSymbol string
}

func (r *LibraryLockerRule) extractLibraryCall(node *reader.RichNode) *LibraryCall {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return nil
	}

	funcSymbol := funcNode.Value
	if !strings.Contains(funcSymbol, "/") {
		return nil
	}

	parts := strings.SplitN(funcSymbol, "/", 2)
	if len(parts) != 2 {
		return nil
	}

	library := parts[0]
	function := parts[1]

	for _, excluded := range r.ExcludedLibraries {
		if library == excluded {
			return nil
		}
	}

	return &LibraryCall{
		Library:    library,
		Function:   function,
		Arguments:  node.Children[1:],
		FullSymbol: funcSymbol,
	}
}

func (r *LibraryLockerRule) isLibraryLocker(params []*reader.RichNode, call *LibraryCall, funcName string) bool {

	if r.hasSimpleParameterDelegation(params, call.Arguments) {
		return true
	}

	if r.hasConfiguredParameterDelegation(params, call.Arguments) {
		return true
	}

	if r.hasReorganizedParameterDelegation(params, call.Arguments) {
		return true
	}

	return false
}

func (r *LibraryLockerRule) hasSimpleParameterDelegation(params []*reader.RichNode, args []*reader.RichNode) bool {
	if len(params) != len(args) {
		return false
	}

	for i, param := range params {
		if param.Type != reader.NodeSymbol || args[i].Type != reader.NodeSymbol {
			return false
		}
		if param.Value != args[i].Value {
			return false
		}
	}

	return true
}

func (r *LibraryLockerRule) hasConfiguredParameterDelegation(params []*reader.RichNode, args []*reader.RichNode) bool {

	if len(args) < len(params) {
		return false
	}

	paramOffset := len(args) - len(params)
	for i, param := range params {
		if param.Type != reader.NodeSymbol {
			return false
		}

		argIndex := paramOffset + i
		arg := args[argIndex]

		if arg.Type != reader.NodeSymbol || param.Value != arg.Value {
			return false
		}
	}

	for i := 0; i < paramOffset; i++ {
		arg := args[i]

		if arg.Type != reader.NodeKeyword &&
			arg.Type != reader.NodeString &&
			arg.Type != reader.NodeNumber &&
			arg.Type != reader.NodeMap &&
			arg.Type != reader.NodeBool {
			return false
		}
	}

	return true
}

func (r *LibraryLockerRule) hasReorganizedParameterDelegation(params []*reader.RichNode, args []*reader.RichNode) bool {
	if len(params) != len(args) {
		return false
	}

	paramMap := make(map[string]bool)
	for _, param := range params {
		if param.Type == reader.NodeSymbol {
			paramMap[param.Value] = false
		}
	}

	for _, arg := range args {
		if arg.Type != reader.NodeSymbol {
			return false
		}
		if _, exists := paramMap[arg.Value]; !exists {
			return false
		}
		paramMap[arg.Value] = true
	}

	for _, used := range paramMap {
		if !used {
			return false
		}
	}

	return true
}

func (r *LibraryLockerRule) formatMessage(funcName string, call *LibraryCall) string {
	return fmt.Sprintf(
		"Function %q appears to be a 'Library Locker' - it merely wraps %q without adding significant value. "+
			"Consider using %q directly or adding meaningful abstraction if the wrapper serves a specific purpose.",
		funcName, call.FullSymbol, call.FullSymbol,
	)
}

func init() {
	defaultRule := &LibraryLockerRule{
		Rule: Rule{
			ID:          "library-locker",
			Name:        "Library Locker",
			Description: "Detects functions that unnecessarily wrap third-party library calls without adding meaningful abstraction. This pattern obscures the library's usage and adds unnecessary indirection.",
			Severity:    SeverityInfo,
		},
		ExcludedLibraries: []string{
			"clojure.core",
			"clojure.string",
			"clojure.set",
			"clojure.walk",
		},
		MinParamCount: 1,
	}

	RegisterRule(defaultRule)
}
