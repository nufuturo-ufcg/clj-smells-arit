package analyzer

import (
	"github.com/thlaurentino/arit/internal/reader"
)

// ExpandMacros traverses the AST and physically mutates nodes to represent their macro-expanded state.
// This is strictly an experimental feature that currently supports -> and ->>.
func ExpandMacros(roots []*reader.RichNode) {
	for _, root := range roots {
		expandNodeRecursively(root)
	}
}

func expandNodeRecursively(node *reader.RichNode) {
	if node == nil {
		return
	}

	// Bottom-up recursion: expand children first before dealing with the parent
	for _, child := range node.Children {
		expandNodeRecursively(child)
	}

	if node.Type == reader.NodeList && len(node.Children) >= 3 && node.Children[0].Type == reader.NodeSymbol {
		head := node.Children[0].Value
		if head == "->" {
			expandThreadFirst(node)
		} else if head == "->>" {
			expandThreadLast(node)
		}
	}
}

func expandThreadFirst(node *reader.RichNode) {
	currentVal := node.Children[1]

	for i := 2; i < len(node.Children); i++ {
		form := node.Children[i]

		if form.Type == reader.NodeSymbol || form.Type == reader.NodeKeyword || form.Type == reader.NodeString {
			// (-> val f) => (f val)
			newList := &reader.RichNode{
				Type:         reader.NodeList,
				InferredType: "List",
				Location:     form.Location,
				Children:     []*reader.RichNode{form, currentVal},
			}
			currentVal = newList
		} else if form.Type == reader.NodeList && len(form.Children) > 0 {
			// (-> val (f arg)) => (f val arg)
			newChildren := make([]*reader.RichNode, 0, len(form.Children)+1)
			newChildren = append(newChildren, form.Children[0]) // f
			newChildren = append(newChildren, currentVal)       // val
			newChildren = append(newChildren, form.Children[1:]...) // arg

			form.Children = newChildren
			currentVal = form
		} else {
			newList := &reader.RichNode{
				Type:         reader.NodeList,
				InferredType: "List",
				Location:     form.Location,
				Children:     []*reader.RichNode{form, currentVal},
			}
			currentVal = newList
		}
	}

	// Mutate the original `node` in-place to become the fully expanded `currentVal`
	node.Type = currentVal.Type
	node.Value = currentVal.Value
	node.Children = currentVal.Children
}

func expandThreadLast(node *reader.RichNode) {
	currentVal := node.Children[1]

	for i := 2; i < len(node.Children); i++ {
		form := node.Children[i]

		if form.Type == reader.NodeSymbol || form.Type == reader.NodeKeyword || form.Type == reader.NodeString {
			// (->> val f) => (f val)
			newList := &reader.RichNode{
				Type:         reader.NodeList,
				InferredType: "List",
				Location:     form.Location,
				Children:     []*reader.RichNode{form, currentVal},
			}
			currentVal = newList
		} else if form.Type == reader.NodeList && len(form.Children) > 0 {
			// (->> val (f arg)) => (f arg val)
			newChildren := make([]*reader.RichNode, 0, len(form.Children)+1)
			newChildren = append(newChildren, form.Children...) // (f arg)
			newChildren = append(newChildren, currentVal)       // val

			form.Children = newChildren
			currentVal = form
		} else {
			newList := &reader.RichNode{
				Type:         reader.NodeList,
				InferredType: "List",
				Location:     form.Location,
				Children:     []*reader.RichNode{form, currentVal},
			}
			currentVal = newList
		}
	}

	node.Type = currentVal.Type
	node.Value = currentVal.Value
	node.Children = currentVal.Children
}
