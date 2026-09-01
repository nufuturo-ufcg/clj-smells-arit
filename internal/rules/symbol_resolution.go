package rules

import "github.com/thlaurentino/arit/internal/reader"

func ResolvedCall(node *reader.RichNode) *reader.SymbolResolution {
	if node == nil || (node.Type != reader.NodeList && node.Type != reader.NodeFnLiteral) || len(node.Children) == 0 {
		return nil
	}
	head := node.Children[0]
	if head == nil || head.Type != reader.NodeSymbol {
		return nil
	}
	return head.Resolution
}

func CallResolvesTo(node *reader.RichNode, canonicalNames ...string) bool {
	resolved := ResolvedCall(node)
	if resolved == nil || resolved.Kind == reader.ResolutionUnresolved || resolved.Kind == reader.ResolutionLocal {
		return false
	}
	for _, canonical := range canonicalNames {
		if resolved.CanonicalName == canonical {
			return true
		}
	}
	return false
}
