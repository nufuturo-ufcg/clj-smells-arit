package reader

import (
	"fmt"

	"github.com/cespare/goclj/parse"
)

func ParseFile(filepath string) (*parse.Tree, error) {
	opts := parse.IncludeNonSemantic
	tree, err := parse.File(filepath, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filepath, err)
	}
	return tree, nil
}

func BuildRichTree(tree *parse.Tree) ([]*RichNode, []*RichNode) {
	if tree == nil {
		return nil, nil
	}

	richRoots := make([]*RichNode, 0, len(tree.Roots))

	for _, rootNode := range tree.Roots {
		if rootNode != nil {
			richNode := buildRichNode(rootNode, true)
			if richNode != nil {
				richRoots = append(richRoots, richNode)
			}
		}
	}

	comments := collectCommentsOptimized(tree, len(tree.Roots)/10+16)
	ApplyTypeHints(richRoots)
	return richRoots, comments
}

func collectCommentsOptimized(tree *parse.Tree, estimatedSize int) []*RichNode {

	comments := make([]*RichNode, 0, estimatedSize)

	var walk func(node parse.Node)
	walk = func(node parse.Node) {
		if node == nil {
			return
		}

		if comment, ok := node.(*parse.CommentNode); ok {
			richComment := buildRichNode(comment, false)
			if richComment != nil {
				comments = append(comments, richComment)
			}
			return
		}

		if children := node.Children(); len(children) > 0 {
			for _, child := range children {
				walk(child)
			}
		}
	}

	for _, root := range tree.Roots {
		walk(root)
	}
	return comments
}

func buildRichNode(node parse.Node, ignoreComments bool) *RichNode {
	if node == nil {
		return nil
	}

	if ignoreComments {
		switch node.(type) {
		case *parse.CommentNode, *parse.NewlineNode:
			return nil
		}
	}

	rNode := &RichNode{
		Location:     extractLocation(node),
		OriginalNode: node,
	}

	switch n := node.(type) {
	case *parse.ListNode:
		rNode.Type = NodeList
		rNode.InferredType = "List"
		if n.Nodes != nil && len(n.Nodes) > 0 {
			rNode.Children = buildRichChildren(n.Nodes, ignoreComments)
		}
	case *parse.VectorNode:
		rNode.Type = NodeVector
		rNode.InferredType = "Vector"
		if n.Nodes != nil && len(n.Nodes) > 0 {
			rNode.Children = buildRichChildren(n.Nodes, ignoreComments)
		}
	case *parse.MapNode:
		rNode.Type = NodeMap
		if n.Nodes != nil {
			rNode.Children = buildRichChildren(n.Nodes, ignoreComments)
		}
		rNode.InferredType = "Map"
	case *parse.SetNode:
		rNode.Type = NodeSet
		if n.Nodes != nil {
			rNode.Children = buildRichChildren(n.Nodes, ignoreComments)
		}
		rNode.InferredType = "Set"

	case *parse.SymbolNode:
		rNode.Type = NodeSymbol
		rNode.Value = n.Val
		rNode.InferredType = "Symbol"
	case *parse.KeywordNode:
		rNode.Type = NodeKeyword
		rNode.Value = n.Val
		rNode.InferredType = "Keyword"
	case *parse.StringNode:
		rNode.Type = NodeString
		rNode.Value = n.Val
		rNode.InferredType = "String"
	case *parse.NumberNode:
		rNode.Type = NodeNumber
		rNode.Value = n.Val
		rNode.InferredType = "Number"
	case *parse.BoolNode:
		rNode.Type = NodeBool
		rNode.Value = fmt.Sprintf("%t", n.Val)
		rNode.InferredType = "Bool"
	case *parse.NilNode:
		rNode.Type = NodeNil
		rNode.Value = "nil"
		rNode.InferredType = "Nil"
	case *parse.RegexNode:
		rNode.Type = NodeRegex
		rNode.Value = n.Val
		rNode.InferredType = "Regex"
	case *parse.CharacterNode:
		rNode.Type = NodeCharacter
		rNode.Value = string(n.Val)
		rNode.InferredType = "Character"

	case *parse.CommentNode:
		if !ignoreComments {
			rNode.Type = NodeComment
			rNode.Value = n.Text
		} else {
			return nil
		}
	case *parse.NewlineNode:
		if !ignoreComments {
			rNode.Type = NodeNewline
		} else {
			return nil
		}
	case *parse.TagNode:
		rNode.Type = NodeTag
		rNode.Value = n.Val

	case *parse.QuoteNode:
		rNode.Type = NodeQuote
		if n.Node != nil {
			if quoted := buildRichNode(n.Node, ignoreComments); quoted != nil {
				rNode.Children = []*RichNode{quoted}
			}
		}
	case *parse.SyntaxQuoteNode:
		rNode.Type = NodeSyntaxQuote
		if n.Node != nil {
			if quoted := buildRichNode(n.Node, ignoreComments); quoted != nil {
				rNode.Children = []*RichNode{quoted}
			}
		}
	case *parse.UnquoteNode:
		rNode.Type = NodeUnquote
		if n.Node != nil {
			if unquoted := buildRichNode(n.Node, ignoreComments); unquoted != nil {
				rNode.Children = []*RichNode{unquoted}
			}
		}
	case *parse.UnquoteSpliceNode:
		rNode.Type = NodeUnquoteSplice
		if n.Node != nil {
			if spliced := buildRichNode(n.Node, ignoreComments); spliced != nil {
				rNode.Children = []*RichNode{spliced}
			}
		}
	case *parse.DerefNode:
		rNode.Type = NodeDeref
		if n.Node != nil {
			if derefd := buildRichNode(n.Node, ignoreComments); derefd != nil {
				rNode.Children = []*RichNode{derefd}
			}
		}
	case *parse.VarQuoteNode:
		rNode.Type = NodeVarQuote
		rNode.Value = n.Val
	case *parse.FnLiteralNode:
		rNode.Type = NodeFnLiteral
		if n.Nodes != nil {
			rNode.Children = buildRichChildren(n.Nodes, ignoreComments)
		}
	case *parse.ReaderCondNode:
		rNode.Type = NodeReaderCond
		if n.Nodes != nil {
			rNode.Children = buildRichChildren(n.Nodes, ignoreComments)
		}
	case *parse.ReaderCondSpliceNode:
		rNode.Type = NodeReaderCondSplice
		if n.Nodes != nil {
			rNode.Children = buildRichChildren(n.Nodes, ignoreComments)
		}
	case *parse.ReaderDiscardNode:
		rNode.Type = NodeReaderDiscard
		if n.Node != nil {
			if discarded := buildRichNode(n.Node, ignoreComments); discarded != nil {
				rNode.Children = []*RichNode{discarded}
			}
		}
	case *parse.ReaderEvalNode:
		rNode.Type = NodeReaderEval
		if n.Node != nil {
			if evalExpr := buildRichNode(n.Node, ignoreComments); evalExpr != nil {
				rNode.Children = []*RichNode{evalExpr}
			}
		}

	case *parse.MetadataNode:
		if n == nil {
			return nil
		}

		children := n.Children()

		var metaHintNodeRaw parse.Node
		var actualNodeRaw parse.Node

		if len(children) == 2 {
			metaHintNodeRaw = children[0]
			actualNodeRaw = children[1]

			if actualNodeRaw != n.Node {

			}
		} else if n.Node != nil {

			actualNodeRaw = n.Node
		} else {
			return nil
		}

		if actualNodeRaw == nil {
			return nil
		}

		richActualNode := buildRichNode(actualNodeRaw, ignoreComments)
		if richActualNode == nil {
			return nil
		}

		if metaHintNodeRaw == nil {
			return richActualNode
		}

		var typeHintFromMeta string
		switch meta := metaHintNodeRaw.(type) {
		case *parse.SymbolNode:
			if meta != nil {
				typeHintFromMeta = meta.Val
			}
		case *parse.TagNode:
			if meta != nil {
				typeHintFromMeta = meta.Val
			}
		case *parse.KeywordNode:

			break
		case *parse.MapNode:
			if meta != nil && meta.Nodes != nil {
				for i := 0; i+1 < len(meta.Nodes); i += 2 {
					if keyNode, okKey := meta.Nodes[i].(*parse.KeywordNode); okKey && keyNode != nil {
						if valueNode, okVal := meta.Nodes[i+1].(*parse.SymbolNode); okVal && valueNode != nil {
							if keyNode.Val == ":tag" {
								typeHintFromMeta = valueNode.Val
								break
							}
						}
					}
				}
			}
		}

		if typeHintFromMeta != "" && richActualNode != nil {
			richActualNode.TypeHint = typeHintFromMeta
		}

		return richActualNode

	default:

		if !ignoreComments {
			rNode.Type = NodeUnknown
		} else {
			rNode.Type = NodeUnknown
		}
	}

	return rNode
}

func buildRichChildren(nodes []parse.Node, ignoreComments bool) []*RichNode {
	richChildren := make([]*RichNode, 0, len(nodes))
	for _, childNode := range nodes {
		richChild := buildRichNode(childNode, ignoreComments)
		if richChild != nil {
			richChildren = append(richChildren, richChild)
		}
	}
	return richChildren
}

func extractLocation(node parse.Node) *Location {
	if node == nil {
		return nil
	}
	pos := node.Position()
	startLine := pos.Line
	startCol := pos.Col
	endLine := startLine
	endCol := startCol + 1
	return &Location{
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     endLine,
		EndColumn:   endCol,
	}
}

func FindTopLevelDefns(tree *parse.Tree) []*parse.ListNode {
	var defns []*parse.ListNode
	for _, root := range tree.Roots {
		if listNode, ok := root.(*parse.ListNode); ok {
			if len(listNode.Nodes) > 1 {
				if symbolNode, ok := listNode.Nodes[0].(*parse.SymbolNode); ok {

					if symbolNode.Val == "defn" || symbolNode.Val == "defn-" {
						defns = append(defns, listNode)
					}
				}
			}
		}
	}
	return defns
}

func ApplyTypeHints(nodes []*RichNode) {

}
