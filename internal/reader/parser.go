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
	discardNext := false

	var pendingMeta []*parse.MetadataNode
	for _, rootNode := range tree.Roots {
		if rootNode != nil {
			if discard, ok := rootNode.(*parse.ReaderDiscardNode); ok {
				// goclj represents a discard followed by a newline as two
				// top-level nodes: the discard marker and the form after it.
				// Keep the marker pending until the next semantic root so the
				// discarded form cannot be analyzed accidentally.
				discardNext = discard.Node == nil
				if discard.Node != nil {
					switch discard.Node.(type) {
					case *parse.CommentNode, *parse.NewlineNode:
						discardNext = true
					}
				}
				continue
			}
			if discardNext {
				switch rootNode.(type) {
				case *parse.CommentNode, *parse.NewlineNode:
					continue
				}
				discardNext = false
				continue
			}
			if meta, ok := rootNode.(*parse.MetadataNode); ok {
				pendingMeta = append(pendingMeta, meta)
				continue
			}
			if isLayoutNode(rootNode) {
				continue
			}
			richNode := buildRichNode(rootNode, true)
			if richNode != nil {
				attachMetadata(richNode, pendingMeta, true)
				richRoots = append(richRoots, richNode)
			}
			pendingMeta = nil
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
		// clj-kondo's own suppression syntax is a discarded form:
		// #_:clj-kondo/ignore or #_{:clj-kondo/ignore [...]}. Expose it as a
		// directive at the discard position so inline suppression honors it.
		if discard, ok := node.(*parse.ReaderDiscardNode); ok {
			if isKondoIgnoreDirective(discard.Node) {
				comments = append(comments, &RichNode{
					Type:     NodeComment,
					Value:    ";; clj-kondo/ignore",
					Location: extractLocation(discard),
				})
			}
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
		// Emulate Reader: prune (comment ...) blocks
		if ignoreComments && len(n.Nodes) > 0 {
			if sym, ok := n.Nodes[0].(*parse.SymbolNode); ok && sym.Val == "comment" {
				return nil
			}
		}

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
		if ignoreComments {
			return nil
		}
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
		richActualNode.Metadata = buildRichNode(metaHintNodeRaw, ignoreComments)

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
	var pendingMeta []*parse.MetadataNode
	discardNext := false
	for i := 0; i < len(nodes); i++ {
		childNode := nodes[i]
		// As in BuildRichTree, a discard followed by a newline or comment is
		// emitted as the marker plus the discarded form as the next sibling;
		// drop that form so code inside `#_` is never analyzed.
		if discard, ok := childNode.(*parse.ReaderDiscardNode); ok {
			switch discard.Node.(type) {
			case nil, *parse.CommentNode, *parse.NewlineNode:
				discardNext = true
				continue
			}
		}
		if discardNext {
			switch childNode.(type) {
			case *parse.CommentNode, *parse.NewlineNode:
				continue
			}
			discardNext = false
			continue
		}
		// goclj emits `^meta form` as a MetadataNode holding only the meta
		// value, followed by the form as the next sibling. Attach the meta to
		// that form instead of emitting the meta value as a separate child,
		// which would turn `(defn ^boolean f ...)` into `(defn boolean f ...)`.
		if meta, ok := childNode.(*parse.MetadataNode); ok {
			pendingMeta = append(pendingMeta, meta)
			continue
		}
		if len(pendingMeta) > 0 && isLayoutNode(childNode) {
			continue
		}
		if len(pendingMeta) > 0 {
			target := buildRichNode(childNode, ignoreComments)
			if _, ok := childNode.(*parse.TagNode); ok && i+1 < len(nodes) {
				if value := buildRichNode(nodes[i+1], ignoreComments); target != nil && value != nil {
					target.Children = []*RichNode{value}
					i++
				}
			}
			if target != nil {
				attachMetadata(target, pendingMeta, ignoreComments)
				richChildren = append(richChildren, target)
			}
			pendingMeta = nil
			continue
		}
		// goclj represents a tagged literal as two sibling nodes: the tag and
		// the tagged form. Keep them together so rules that consume forms in
		// pairs (notably `case`) do not mistake the tagged value for another
		// clause.
		if _, ok := childNode.(*parse.TagNode); ok && i+1 < len(nodes) {
			tagged := buildRichNode(childNode, ignoreComments)
			value := buildRichNode(nodes[i+1], ignoreComments)
			if tagged != nil && value != nil {
				tagged.Children = []*RichNode{value}
				richChildren = append(richChildren, tagged)
				i++
				continue
			}
		}
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
	metadataTag := func(node *RichNode) string {
		if node == nil || node.Type != NodeMap {
			return ""
		}
		for i := 0; i+1 < len(node.Children); i += 2 {
			if node.Children[i] != nil && node.Children[i].Type == NodeKeyword && node.Children[i].Value == ":tag" &&
				node.Children[i+1] != nil && node.Children[i+1].Type == NodeSymbol {
				return node.Children[i+1].Value
			}
		}
		return ""
	}
	var apply func(*RichNode)
	apply = func(node *RichNode) {
		if node == nil {
			return
		}
		if node.Type == NodeTag && len(node.Children) == 1 && node.Children[0] != nil {
			node.Children[0].TypeHint = node.Value
		}
		for i, child := range node.Children {
			if tag := metadataTag(child); tag != "" && i+1 < len(node.Children) && node.Children[i+1] != nil {
				node.Children[i+1].TypeHint = tag
			}
		}
		for _, child := range node.Children {
			apply(child)
		}
	}
	for _, node := range nodes {
		apply(node)
	}
}

func isLayoutNode(node parse.Node) bool {
	switch node.(type) {
	case *parse.CommentNode, *parse.NewlineNode:
		return true
	}
	return false
}

// attachMetadata records reader metadata on the form it annotates. Stacked
// metadata (^:private ^long f) is chained through each meta node's Metadata
// field so no marker is lost; a symbol, string, or {:tag T} value provides
// the type hint.
func attachMetadata(target *RichNode, metas []*parse.MetadataNode, ignoreComments bool) {
	tail := target
	for tail.Metadata != nil {
		tail = tail.Metadata
	}
	for _, meta := range metas {
		if meta == nil || meta.Node == nil {
			continue
		}
		built := buildRichNode(meta.Node, ignoreComments)
		if built == nil {
			continue
		}
		tail.Metadata = built
		for tail.Metadata != nil {
			tail = tail.Metadata
		}
		if hint := metadataTypeHint(meta.Node); hint != "" {
			target.TypeHint = hint
		}
	}
}

func metadataTypeHint(node parse.Node) string {
	switch meta := node.(type) {
	case *parse.SymbolNode:
		return meta.Val
	case *parse.StringNode:
		return meta.Val
	case *parse.MapNode:
		for i := 0; i+1 < len(meta.Nodes); i += 2 {
			if key, ok := meta.Nodes[i].(*parse.KeywordNode); ok && key.Val == ":tag" {
				if value, ok := meta.Nodes[i+1].(*parse.SymbolNode); ok {
					return value.Val
				}
			}
		}
	}
	return ""
}

func isKondoIgnoreDirective(node parse.Node) bool {
	switch directive := node.(type) {
	case *parse.KeywordNode:
		return directive.Val == ":clj-kondo/ignore"
	case *parse.MapNode:
		// {:clj-kondo/ignore [:linter ...]} silences only the listed clj-kondo
		// linters, which do not correspond to ARIT rules; only a value that
		// is not a linter list ignores everything.
		for i := 0; i+1 < len(directive.Nodes); i += 2 {
			if key, ok := directive.Nodes[i].(*parse.KeywordNode); ok && key.Val == ":clj-kondo/ignore" {
				switch directive.Nodes[i+1].(type) {
				case *parse.VectorNode, *parse.ListNode, *parse.SetNode:
					return false
				}
				return true
			}
		}
	}
	return false
}
