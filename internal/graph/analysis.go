package graph

func (g *Graph) ReachableFrom(startID string, kind EdgeKind) []string {
	if g.Node(startID) == nil {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	var visit func(id string)
	visit = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
		for _, e := range g.EdgesFrom(id) {
			if e.Kind != kind {
				continue
			}
			visit(e.To)
		}
	}
	visit(startID)
	return out
}

func (g *Graph) ReachingTo(targetID string, kind EdgeKind) []string {
	if g.Node(targetID) == nil {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	var visit func(id string)
	visit = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
		for _, e := range g.EdgesTo(id) {
			if e.Kind != kind {
				continue
			}
			visit(e.From)
		}
	}
	visit(targetID)
	return out
}


func (g *Graph) ImpactSet(id string) []string {
	reaching := g.ReachingTo(id, EdgeCalls)
	result := make([]string, 0, len(reaching))
	for _, nid := range reaching {
		if nid != id {
			result = append(result, nid)
		}
	}
	return result
}

func (g *Graph) EntryPoints() []string {
	var out []string
	for id, n := range g.Nodes {
		if n.Kind != KindFunction {
			continue
		}
		hasCaller := false
		for _, e := range g.EdgesTo(id) {
			if e.Kind == EdgeCalls {
				hasCaller = true
				break
			}
		}
		if !hasCaller {
			out = append(out, id)
		}
	}
	return out
}

func (g *Graph) DeadCode(entryIDs []string) []string {
	entries := entryIDs
	if len(entries) == 0 {
		entries = g.EntryPoints()
	}
	reachable := make(map[string]bool)
	for _, eid := range entries {
		for _, id := range g.ReachableFrom(eid, EdgeCalls) {
			reachable[id] = true
		}
	}
	var out []string
	for id, n := range g.Nodes {
		if n.Kind != KindFunction {
			continue
		}
		if !reachable[id] {
			out = append(out, id)
		}
	}
	return out
}

func (g *Graph) NamespaceLayers() map[string]int {
	layers := make(map[string]int)
	for id, n := range g.Nodes {
		if n.Kind == KindNamespace {
			layers[id] = -1 // unset
		}
	}
	for changed := true; changed; {
		changed = false
		for id, n := range g.Nodes {
			if n.Kind != KindNamespace {
				continue
			}
			if layers[id] >= 0 {
				continue
			}
			maxDep := -1
			allDepSet := true
			for _, e := range g.EdgesFrom(id) {
				if e.Kind != EdgeRequires {
					continue
				}
				depLayer, ok := layers[e.To]
				if !ok {
					continue
				}
				if depLayer < 0 {
					allDepSet = false
					break
				}
				if depLayer > maxDep {
					maxDep = depLayer
				}
			}
			if allDepSet && (maxDep < 0 || layers[id] < 0) {
				layers[id] = maxDep + 1
				changed = true
			}
		}
	}
	for id := range layers {
		if layers[id] < 0 {
			layers[id] = 0
		}
	}
	return layers
}

func (g *Graph) LongestCallPath() []string {
	var best []string
	cur := make([]string, 0, 32)
	used := make(map[string]bool)

	var dfs func(id string)
	dfs = func(id string) {
		if used[id] {
			return
		}
		used[id] = true
		cur = append(cur, id)
		if len(cur) > len(best) {
			best = make([]string, len(cur))
			copy(best, cur)
		}
		for _, e := range g.EdgesFrom(id) {
			if e.Kind != EdgeCalls {
				continue
			}
			dfs(e.To)
		}
		cur = cur[:len(cur)-1]
		used[id] = false
	}

	for id, n := range g.Nodes {
		if n.Kind != KindFunction {
			continue
		}
		dfs(id)
	}
	return best
}

func (g *Graph) Subgraph(keep map[string]bool) *Graph {
	if len(keep) == 0 {
		return NewGraph()
	}
	out := NewGraph()
	for id, n := range g.Nodes {
		if !keep[id] {
			continue
		}
		out.AddNode(&Node{ID: n.ID, Kind: n.Kind, Filepath: n.Filepath, Line: n.Line})
	}
	for _, e := range g.Edges() {
		if keep[e.From] && keep[e.To] {
			out.AddEdge(&Edge{From: e.From, To: e.To, Kind: e.Kind, Location: e.Location})
		}
	}
	return out
}

func (g *Graph) SubgraphByEdgeKinds(kinds []EdgeKind) *Graph {
	set := make(map[EdgeKind]bool)
	for _, k := range kinds {
		set[k] = true
	}
	out := NewGraph()
	for _, e := range g.Edges() {
		if !set[e.Kind] {
			continue
		}
		if from := g.Node(e.From); from != nil {
			out.AddNode(&Node{ID: from.ID, Kind: from.Kind, Filepath: from.Filepath, Line: from.Line})
		}
		if to := g.Node(e.To); to != nil {
			out.AddNode(&Node{ID: to.ID, Kind: to.Kind, Filepath: to.Filepath, Line: to.Line})
		}
		out.AddEdge(&Edge{From: e.From, To: e.To, Kind: e.Kind, Location: e.Location})
	}
	return out
}
