package reader

import (
	"strings"

	"github.com/cespare/goclj/parse"
)

type NodeType string

const (
	NodeList   NodeType = "List"
	NodeVector NodeType = "Vector"
	NodeMap    NodeType = "Map"
	NodeSet    NodeType = "Set"

	NodeSymbol    NodeType = "Symbol"
	NodeKeyword   NodeType = "Keyword"
	NodeString    NodeType = "String"
	NodeNumber    NodeType = "Number"
	NodeBool      NodeType = "Bool"
	NodeNil       NodeType = "Nil"
	NodeCharacter NodeType = "Character"
	NodeComment   NodeType = "Comment"
	NodeMetadata  NodeType = "Metadata"
	NodeTag       NodeType = "Tag"
	NodeRegex     NodeType = "Regex"

	NodeQuote            NodeType = "Quote"
	NodeSyntaxQuote      NodeType = "SyntaxQuote"
	NodeUnquote          NodeType = "Unquote"
	NodeUnquoteSplice    NodeType = "UnquoteSplice"
	NodeDeref            NodeType = "Deref"
	NodeVarQuote         NodeType = "VarQuote"
	NodeFnLiteral        NodeType = "FnLiteral"
	NodeReaderCond       NodeType = "ReaderCond"
	NodeReaderCondSplice NodeType = "ReaderCondSplice"
	NodeReaderDiscard    NodeType = "ReaderDiscard"
	NodeReaderEval       NodeType = "ReaderEval"

	NodeNewline NodeType = "Newline"
	NodeUnknown NodeType = "Unknown"
)

type Location struct {
	StartLine   int `json:"start_line"`
	StartColumn int `json:"start_col"`
	EndLine     int `json:"end_line"`
	EndColumn   int `json:"end_col"`
}

type RichNode struct {
	Type     NodeType    `json:"type"`
	Value    string      `json:"value,omitempty"`
	Location *Location   `json:"location,omitempty"`
	Children []*RichNode `json:"children,omitempty"`
	Metadata *RichNode   `json:"metadata,omitempty"`
	Comments []*RichNode `json:"comments,omitempty"`
	TypeHint string      `json:"type_hint,omitempty"`

	ResolvedDefinition *RichNode `json:"-"`

	InferredType string `json:"inferred_type,omitempty"`

	OriginalNode parse.Node `json:"-"`

	Scope     interface{}
	SymbolRef interface{}
}

func CountFunctionParameters(paramsNode *RichNode) int {
	if paramsNode == nil || paramsNode.Type != NodeVector {
		return 0
	}

	count := 0
	foundVariadic := false

	for i, param := range paramsNode.Children {
		if param.Type == NodeSymbol {

			if param.Value == "_" ||
				strings.HasPrefix(param.Value, ".") ||
				strings.Contains(param.Value, "/") {
				continue
			}

			if param.Value == "&" {
				foundVariadic = true
				continue
			}

			if foundVariadic {
				continue
			}
		}

		if !foundVariadic {
			count++
		}
	}
	return count
}

func IsVariadicMarker(node *RichNode) bool {
	return node != nil && node.Type == NodeSymbol && node.Value == "&"
}
