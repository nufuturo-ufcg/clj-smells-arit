package graph

import (
	"encoding/json"
	"fmt"
	"strings"
)

type NodeKind string

const (
	KindNamespace NodeKind = "namespace"
	KindFunction  NodeKind = "function"
	KindVar       NodeKind = "var"
)

type EdgeKind string

const (
	EdgeRequires EdgeKind = "requires" 
	EdgeRefers   EdgeKind = "refers"   
	EdgeCalls    EdgeKind = "calls"    
	EdgeReads    EdgeKind = "reads"   
)

type Location struct {
	Filepath string `json:"filepath,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"col,omitempty"`
}

type Node struct {
	ID       string   `json:"id"`
	Kind     NodeKind `json:"kind"`
	Filepath string   `json:"filepath,omitempty"`
	Line     int      `json:"line,omitempty"`
}

type Edge struct {
	From     string   `json:"from"`               
	To       string   `json:"to"`                 
	Kind     EdgeKind `json:"kind"`               
	Location Location `json:"location,omitempty"` 
}

type Graph struct {
	Nodes map[string]*Node `json:"nodes"`

	edges []*Edge

	edgesByFrom map[string][]*Edge

	edgesByTo map[string][]*Edge
}

func NewGraph() *Graph {
	return &Graph{
		Nodes:       make(map[string]*Node),
		edges:       nil,
		edgesByFrom: make(map[string][]*Edge),
		edgesByTo:   make(map[string][]*Edge),
	}
}

func (g *Graph) AddNode(n *Node) {
	if n == nil || n.ID == "" {
		return
	}
	g.Nodes[n.ID] = n
}

func (g *Graph) Node(id string) *Node {
	return g.Nodes[id]
}

func (g *Graph) AddEdge(e *Edge) {
	if e == nil || e.From == "" || e.To == "" {
		return
	}
	g.edges = append(g.edges, e)
	g.edgesByFrom[e.From] = append(g.edgesByFrom[e.From], e)
	g.edgesByTo[e.To] = append(g.edgesByTo[e.To], e)
}

func (g *Graph) Edges() []*Edge {
	if len(g.edges) == 0 {
		return nil
	}
	out := make([]*Edge, len(g.edges))
	copy(out, g.edges)
	return out
}

func (g *Graph) EdgesFrom(id string) []*Edge {
	return g.edgesByFrom[id]
}

func (g *Graph) EdgesTo(id string) []*Edge {
	return g.edgesByTo[id]
}

func (g *Graph) NodeCount() int {
	return len(g.Nodes)
}

func (g *Graph) EdgeCount() int {
	return len(g.edges)
}

type exportData struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

func (g *Graph) ToJSON() ([]byte, error) {
	nodes := make([]*Node, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		nodes = append(nodes, n)
	}
	return json.Marshal(exportData{Nodes: nodes, Edges: g.Edges()})
}

func dotEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func (g *Graph) ToDOT() string {
	var b strings.Builder
	b.WriteString("digraph deps {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=plaintext];\n\n")
	for _, n := range g.Nodes {
		id := dotEscape(n.ID)
		label := dotEscape(n.ID)
		var color string
		switch n.Kind {
		case KindNamespace:
			color = "#e1f5fe"
		case KindFunction:
			color = "#e8f5e9"
		case KindVar:
			color = "#fff3e0"
		default:
			color = "#f5f5f5"
		}
		b.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" style=filled fillcolor=\"%s\"];\n", id, label, color))
	}
	b.WriteString("\n")
	for _, e := range g.Edges() {
		from := dotEscape(e.From)
		to := dotEscape(e.To)
		label := string(e.Kind)
		b.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\"];\n", from, to, dotEscape(label)))
	}
	b.WriteString("}\n")
	return b.String()
}
