package analyzer

import (
	"strings"

	"github.com/thlaurentino/arit/internal/graph"
	"github.com/thlaurentino/arit/internal/reader"
)

type UnifiedGraphInput struct {
	Filepath string
	Result   AnalysisResult
}

type nsGraphParsed struct {
	aliasToNs map[string]string            
	refers    []struct{ Ns, Sym string }  
}

func getNsNode(roots []*reader.RichNode) *reader.RichNode {
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) < 2 {
			continue
		}
		head := root.Children[0]
		if head.Type != reader.NodeSymbol && head.Type != reader.NodeKeyword {
			continue
		}
		if normKeyword(head.Value) == "ns" {
			return root
		}
	}
	return nil
}

func normKeyword(s string) string {
	if strings.HasPrefix(s, ":") {
		return s[1:]
	}
	return s
}

func parseNsForGraph(nsNode *reader.RichNode) (nsName string, aliasToNs map[string]string, refers []struct{ Ns, Sym string }) {
	if nsNode == nil || nsNode.Type != reader.NodeList || len(nsNode.Children) < 2 {
		return "", nil, nil
	}
	if len(nsNode.Children) > 1 && nsNode.Children[1].Type == reader.NodeSymbol {
		nsName = nsNode.Children[1].Value
	}
	aliasToNs = make(map[string]string)

	for i := 2; i < len(nsNode.Children); i++ {
		clause := nsNode.Children[i]
		if clause.Type != reader.NodeList || len(clause.Children) == 0 {
			continue
		}
		first := clause.Children[0]
		if first.Type != reader.NodeKeyword && first.Type != reader.NodeSymbol {
			continue
		}
		key := normKeyword(first.Value)
		switch key {
		case "require":
			for j := 1; j < len(clause.Children); j++ {
				spec := clause.Children[j]
				if spec.Type != reader.NodeVector || len(spec.Children) == 0 {
					continue
				}
				nsSym := spec.Children[0]
				if nsSym.Type != reader.NodeSymbol {
					continue
				}
				fullNs := nsSym.Value
				var alias string
				for k := 1; k+1 < len(spec.Children); k += 2 {
					optKey := spec.Children[k]
					if optKey.Type != reader.NodeKeyword && optKey.Type != reader.NodeSymbol {
						continue
					}
					if normKeyword(optKey.Value) != "as" && normKeyword(optKey.Value) != "refer" {
						continue
					}
					optVal := spec.Children[k+1]
					if normKeyword(optKey.Value) == "as" && optVal.Type == reader.NodeSymbol {
						alias = optVal.Value
						aliasToNs[alias] = fullNs
					}
					if normKeyword(optKey.Value) == "refer" && optVal.Type == reader.NodeVector {
						for _, symNode := range optVal.Children {
							if symNode.Type == reader.NodeSymbol {
								refers = append(refers, struct{ Ns, Sym string }{fullNs, symNode.Value})
							}
						}
					}
				}
				if alias == "" {
					aliasToNs[fullNs] = fullNs
				}
			}
		case "import":
			for j := 1; j < len(clause.Children); j++ {
				imp := clause.Children[j]
				if imp.Type == reader.NodeSymbol {
					full := imp.Value
					last := strings.LastIndex(full, ".")
					if last > 0 && last < len(full)-1 {
						refers = append(refers, struct{ Ns, Sym string }{full, full[last+1:]})
					}
				}
			}
		}
	}
	return nsName, aliasToNs, refers
}

func BuildUnifiedGraph(inputs []UnifiedGraphInput) *graph.Graph {
	g := graph.NewGraph()
	aliasMaps := make([]map[string]string, 0, len(inputs))

	for _, in := range inputs {
		filepath := in.Filepath
		res := in.Result
		ns := res.Namespace
		var aliasToNs map[string]string

		nsNode := getNsNode(res.RichRoots)
		if nsNode != nil {
			parsedNs, parsedAlias, parsedRefers := parseNsForGraph(nsNode)
			if parsedNs != "" {
				ns = parsedNs
			}
			aliasToNs = parsedAlias
			seenNs := make(map[string]bool)
			for _, fullNs := range parsedAlias {
				if seenNs[fullNs] {
					continue
				}
				seenNs[fullNs] = true
				g.AddNode(&graph.Node{ID: fullNs, Kind: graph.KindNamespace})
				g.AddEdge(&graph.Edge{From: ns, To: fullNs, Kind: graph.EdgeRequires})
			}
			for _, r := range parsedRefers {
				targetID := r.Ns + "/" + r.Sym
				g.AddNode(&graph.Node{ID: targetID, Kind: graph.KindVar})
				g.AddEdge(&graph.Edge{From: ns, To: targetID, Kind: graph.EdgeRefers})
			}
		}
		if aliasToNs == nil {
			aliasToNs = make(map[string]string)
			for _, a := range res.Aliases {
				aliasToNs[a.Alias] = a.FullNamespace
			}
		}
		aliasMaps = append(aliasMaps, aliasToNs)

		if ns == "" {
			continue
		}
		g.AddNode(&graph.Node{ID: ns, Kind: graph.KindNamespace, Filepath: filepath})

		collectDefnsAndDefs(res.RichRoots, ns, filepath, g)
	}

	for i, in := range inputs {
		res := in.Result
		ns := res.Namespace
		if ns == "" {
			continue
		}
		aliasToNs := aliasMaps[i]
		if aliasToNs == nil {
			aliasToNs = make(map[string]string)
			for _, a := range res.Aliases {
				aliasToNs[a.Alias] = a.FullNamespace
			}
		}
		nsNode := getNsNode(res.RichRoots)
		if nsNode != nil {
			parsedNs, parsedAlias, _ := parseNsForGraph(nsNode)
			if parsedNs != "" {
				ns = parsedNs
			}
			if len(parsedAlias) > 0 {
				aliasToNs = parsedAlias
			}
		}
		collectCallsAndReads(g, res.RichRoots, ns, in.Filepath, aliasToNs)
	}

	return g
}

func collectDefnsAndDefs(roots []*reader.RichNode, ns, filepath string, g *graph.Graph) {
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) < 2 {
			continue
		}
		head := root.Children[0]
		if head.Type != reader.NodeSymbol {
			continue
		}
		nameNode := root.Children[1]
		if nameNode == nil || nameNode.Type != reader.NodeSymbol {
			continue
		}
		name := nameNode.Value
		line := 0
		if root.Location != nil {
			line = root.Location.StartLine
		}
		id := ns + "/" + name
		switch head.Value {
		case "defn", "defn-":
			g.AddNode(&graph.Node{ID: id, Kind: graph.KindFunction, Filepath: filepath, Line: line})
		case "def", "defonce":
			g.AddNode(&graph.Node{ID: id, Kind: graph.KindVar, Filepath: filepath, Line: line})
		}
	}
}

func collectCallsAndReads(g *graph.Graph, roots []*reader.RichNode, currentNs, filepath string, aliasToNs map[string]string) {
	var visit func(node *reader.RichNode, callerID string, isCallPosition bool)
	visit = func(node *reader.RichNode, callerID string, isCallPosition bool) {
		if node == nil {
			return
		}

		nextCaller := callerID
		bodyStart := -1

		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
			headVal := node.Children[0].Value
			switch headVal {
			case "defn", "defn-":
				if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
					nextCaller = currentNs + "/" + node.Children[1].Value
					bodyStart = defnBodyStartIndex(node)
				}
			case "def", "defonce":
				if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {
					nextCaller = currentNs + "/" + node.Children[1].Value
					bodyStart = 2
				}
			case "fn":
				bodyStart = fnBodyStartIndex(node)
			case "let", "loop":
				for i, child := range node.Children {
					if i <= 1 {
						visit(child, callerID, false)
					} else {
						visit(child, nextCaller, i == 2)
					}
				}
				return
			}
		}

		if bodyStart >= 0 && callerID != "" {
			for i := bodyStart; i < len(node.Children); i++ {
				visit(node.Children[i], nextCaller, true)
			}
			return
		}

		if node.Type == reader.NodeList && len(node.Children) > 0 {
			symNode := node.Children[0]
			if symNode.Type == reader.NodeSymbol && callerID != "" && !isSpecialForm(symNode.Value) {
				targetID := resolveTargetID(symNode, currentNs, aliasToNs)
				if targetID != "" && targetID != callerID {
					ensureNodeExists(g, targetID, graph.KindFunction)
					line := 0
					if node.Location != nil {
						line = node.Location.StartLine
					}
					g.AddEdge(&graph.Edge{
						From: callerID, To: targetID, Kind: graph.EdgeCalls,
						Location: graph.Location{Filepath: filepath, Line: line},
					})
				}
			}
		}

		if !isCallPosition && node.Type == reader.NodeSymbol && callerID != "" {
			targetID := resolveTargetID(node, currentNs, aliasToNs)
			if targetID != "" && targetID != callerID {
				ensureNodeExists(g, targetID, graph.KindVar)
				line := 0
				if node.Location != nil {
					line = node.Location.StartLine
				}
				g.AddEdge(&graph.Edge{
					From: callerID, To: targetID, Kind: graph.EdgeReads,
					Location: graph.Location{Filepath: filepath, Line: line},
				})
			}
		}

		for i, child := range node.Children {
			isChildCallPos := node.Type == reader.NodeList && len(node.Children) > 0 && i == 0
			visit(child, nextCaller, isChildCallPos)
		}
	}

	for _, root := range roots {
		visit(root, "", false)
	}
}

func defnBodyStartIndex(node *reader.RichNode) int {
	idx := 2
	if len(node.Children) <= idx {
		return idx
	}
	if node.Children[idx].Type == reader.NodeString {
		idx++
	}
	if len(node.Children) > idx && node.Children[idx].Type == reader.NodeMap {
		idx++
	}
	if len(node.Children) > idx && (node.Children[idx].Type == reader.NodeVector || node.Children[idx].Type == reader.NodeList) {
		idx++
	}
	return idx
}

func fnBodyStartIndex(node *reader.RichNode) int {
	idx := 1
	if len(node.Children) > idx && node.Children[idx].Type == reader.NodeSymbol {
		idx++
	}
	if len(node.Children) > idx && (node.Children[idx].Type == reader.NodeVector || node.Children[idx].Type == reader.NodeList) {
		idx++
	}
	return idx
}

var specialForms = map[string]bool{
	"defn": true, "defn-": true, "def": true, "defonce": true, "fn": true,
	"let": true, "loop": true, "if": true, "when": true, "cond": true, "case": true,
	"do": true, "recur": true, "quote": true, "var": true, "set!": true,
	"try": true, "catch": true, "finally": true, "new": true, "throw": true,
	".": true, "ns": true,
}

func isSpecialForm(name string) bool { return specialForms[name] }

func ensureNodeExists(g *graph.Graph, id string, kind graph.NodeKind) {
	if g.Node(id) != nil {
		return
	}
	g.AddNode(&graph.Node{ID: id, Kind: kind})
}

func resolveTargetID(symNode *reader.RichNode, currentNs string, aliasToNs map[string]string) string {
	if symNode == nil || symNode.Type != reader.NodeSymbol {
		return ""
	}
	name := symNode.Value
	if name == "" {
		return ""
	}

	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) != 2 {
			return ""
		}
		aliasOrNs := parts[0]
		sym := parts[1]
		if fullNs, ok := aliasToNs[aliasOrNs]; ok {
			return fullNs + "/" + sym
		}
		return aliasOrNs + "/" + sym
	}

	if symNode.SymbolRef != nil {
		if info, ok := symNode.SymbolRef.(*SymbolInfo); ok && info.OriginNamespace != "" {
			return info.OriginNamespace + "/" + info.Name
		}
	}
	return currentNs + "/" + name
}
