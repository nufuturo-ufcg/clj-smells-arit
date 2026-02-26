package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thlaurentino/arit/internal/analyzer"
	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/graph"
)

var depsFormat string
var depsFrom, depsReachableFrom, depsReachingTo, depsImpact string
var depsEntries, depsDeadCode bool
var depsLayers, depsLongestPath bool
var depsDeadCodeEntries string
var depsList bool
var depsEdges string

var depsCmd = &cobra.Command{
	Use:   "deps [file-or-dir...]",
	Short: "Build and output the unified dependency graph",
	Long: `Build the dependency graph from Clojure files and output as HTML.
Use the same file/directory arguments as the main arit command.

Analysis options (use -o to get HTML graph; without -o prints node list; use --list to force list):
  --reachable-from ID   subgraph reachable from ID (calls)
  --reaching-to ID      subgraph of nodes that can reach ID (callers)
  --impact ID           subgraph of callers of ID (impacted if you change ID)
  --entries             subgraph of entry points and what they call
  --dead-code           subgraph of functions not reachable from any entry
  --longest-path        subgraph: the longest call chain

Always list (no graph): --layers

Filter for graph output:
  --from ID             only subgraph reachable from ID
  --edges TYPES         only show these edge types (comma-separated: requires, refers, calls, reads)
`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDeps,
}

func init() {
	rootCmd.AddCommand(depsCmd)
	depsCmd.Flags().StringVarP(&depsFormat, "output", "o", "html", "Output HTML graph (grafo completo ou subgrafo da análise)")
	depsCmd.Flags().StringVar(&depsFrom, "from", "", "Output only subgraph reachable from this node ID (e.g. myapp.core/main)")
	depsCmd.Flags().StringVar(&depsReachableFrom, "reachable-from", "", "List node IDs reachable from this ID (calls)")
	depsCmd.Flags().StringVar(&depsReachingTo, "reaching-to", "", "List node IDs that can reach this ID (callers)")
	depsCmd.Flags().StringVar(&depsImpact, "impact", "", "List node IDs impacted if this node changes (transitive callers)")
	depsCmd.Flags().BoolVar(&depsEntries, "entries", false, "List entry points (functions with no callers)")
	depsCmd.Flags().BoolVar(&depsDeadCode, "dead-code", false, "List functions not reachable from any entry")
	depsCmd.Flags().StringVar(&depsDeadCodeEntries, "dead-code-entries", "", "Comma-separated entry IDs for dead-code (default: auto-detect entries)")
	depsCmd.Flags().BoolVar(&depsLayers, "layers", false, "Print namespace layers (0 = no internal requires)")
	depsCmd.Flags().BoolVar(&depsLongestPath, "longest-path", false, "Print longest call chain (node IDs)")
	depsCmd.Flags().BoolVar(&depsList, "list", false, "With an analysis flag: print node list instead of graph (default when -o is set: output graph)")
	depsCmd.Flags().StringVar(&depsEdges, "edges", "", "Show only these edge types in the graph (comma-separated: requires, refers, calls, reads)")
}

func runDeps(cmd *cobra.Command, args []string) error {
	filesToAnalyze := []string{}
	for _, arg := range args {
		fileInfo, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing %q: %v\n", arg, err)
			continue
		}
		if fileInfo.IsDir() {
			cljFiles, err := findClojureFiles(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding Clojure files in %q: %v\n", arg, err)
				continue
			}
			filesToAnalyze = append(filesToAnalyze, cljFiles...)
		} else {
			ext := strings.ToLower(filepath.Ext(arg))
			if ext == ".clj" || ext == ".cljs" || ext == ".cljc" {
				filesToAnalyze = append(filesToAnalyze, arg)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: skipping non-Clojure file %q\n", arg)
			}
		}
	}
	if len(filesToAnalyze) == 0 {
		return fmt.Errorf("no Clojure files found to analyze")
	}
	sort.Strings(filesToAnalyze)

	configDir := "."
	if len(filesToAnalyze) > 0 {
		firstFileAbs, err := filepath.Abs(filesToAnalyze[0])
		if err == nil {
			parentDir := filepath.Dir(firstFileAbs)
			for parentDir != "/" && parentDir != "." {
				gitPath := filepath.Join(parentDir, ".git")
				modPath := filepath.Join(parentDir, "go.mod")
				gitInfo, gitErr := os.Stat(gitPath)
				modInfo, modErr := os.Stat(modPath)
				if (gitErr == nil && gitInfo.IsDir()) || (modErr == nil && modInfo != nil && !modInfo.IsDir()) {
					configDir = parentDir
					break
				}
				parentDir = filepath.Dir(parentDir)
			}
			if configDir == "." {
				configDir = filepath.Dir(firstFileAbs)
			}
		}
	}
	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		cfg = &config.Config{
			EnabledRules: make(map[string]bool),
			RuleConfig:   make(map[string]config.RuleSettings),
		}
	}

	inputs := make([]analyzer.UnifiedGraphInput, 0, len(filesToAnalyze))
	for _, filePath := range filesToAnalyze {
		result, analyzeErr := analyzer.AnalyzeFile(filePath, cfg)
		if analyzeErr != nil {
			if verboseFlag {
				fmt.Fprintf(os.Stderr, "Warning: skipping %q: %v\n", filePath, analyzeErr)
			}
			continue
		}
		inputs = append(inputs, analyzer.UnifiedGraphInput{Filepath: filePath, Result: result})
	}
	if len(inputs) == 0 {
		return fmt.Errorf("no files were analyzed successfully")
	}

	g := analyzer.BuildUnifiedGraph(inputs)
	if verboseFlag {
		fmt.Fprintf(os.Stderr, "Graph: %d nodes, %d edges\n", g.NodeCount(), g.EdgeCount())
	}

	if depsLayers {
		layers := g.NamespaceLayers()
		nsList := make([]string, 0, len(layers))
		for id := range layers {
			nsList = append(nsList, id)
		}
		sort.Strings(nsList)
		for _, id := range nsList {
			fmt.Printf("%s %d\n", id, layers[id])
		}
		return nil
	}

	wantGraph := !depsList && cmd.Flags().Changed("output")
	var highlightIDs []string 
	var analysisMessage string

	if depsReachableFrom != "" {
		ids := g.ReachableFrom(depsReachableFrom, graph.EdgeCalls)
		if !wantGraph {
			for _, id := range ids {
				fmt.Println(id)
			}
			if len(ids) == 0 {
				fmt.Println(depsReachableFrom)
			}
			return nil
		}
		highlightIDs = []string{depsReachableFrom}
		keep := make(map[string]bool)
		for _, id := range ids {
			keep[id] = true
		}
		keep[depsReachableFrom] = true
		if g.Node(depsReachableFrom) == nil {
			emptyG := graph.NewGraph()
			emptyG.AddNode(&graph.Node{ID: depsReachableFrom, Kind: graph.KindFunction})
			g = emptyG
		} else {
			g = g.Subgraph(keep)
		}
	} else if depsReachingTo != "" {
		ids := g.ReachingTo(depsReachingTo, graph.EdgeCalls)
		if !wantGraph {
			for _, id := range ids {
				fmt.Println(id)
			}
			return nil
		}
		highlightIDs = []string{depsReachingTo}
		keep := make(map[string]bool)
		for _, id := range ids {
			keep[id] = true
		}
		g = g.Subgraph(keep)
	} else if depsImpact != "" {
		ids := g.ImpactSet(depsImpact)
		if !wantGraph {
			for _, id := range ids {
				fmt.Println(id)
			}
			return nil
		}
		highlightIDs = []string{depsImpact}
		keep := make(map[string]bool)
		for _, id := range ids {
			keep[id] = true
		}
		keep[depsImpact] = true
		g = g.Subgraph(keep)
	} else if depsEntries {
		ids := g.EntryPoints()
		if !wantGraph {
			for _, id := range ids {
				fmt.Println(id)
			}
			return nil
		}
		highlightIDs = ids
		keep := make(map[string]bool)
		for _, id := range ids {
			keep[id] = true
			for _, reachID := range g.ReachableFrom(id, graph.EdgeCalls) {
				keep[reachID] = true
			}
		}
		g = g.Subgraph(keep)
	} else if depsDeadCode {
		var entries []string
		if depsDeadCodeEntries != "" {
			for _, s := range strings.Split(depsDeadCodeEntries, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					entries = append(entries, s)
				}
			}
		}
		ids := g.DeadCode(entries)
		if !wantGraph {
			for _, id := range ids {
				fmt.Println(id)
			}
			return nil
		}
		highlightIDs = ids
		if len(ids) > 0 {
			keep := make(map[string]bool)
			for _, id := range ids {
				keep[id] = true
			}
			g = g.Subgraph(keep)
		} else {
			analysisMessage = "No dead code found"
		}
	} else if depsLongestPath {
		path := g.LongestCallPath()
		if !wantGraph {
			for _, id := range path {
				fmt.Println(id)
			}
			return nil
		}
		highlightIDs = path
		keep := make(map[string]bool)
		for _, id := range path {
			keep[id] = true
		}
		sub := g.Subgraph(keep)
		newG := graph.NewGraph()
		for _, id := range path {
			if n := sub.Node(id); n != nil {
				newG.AddNode(&graph.Node{ID: n.ID, Kind: n.Kind, Filepath: n.Filepath, Line: n.Line})
			}
		}
		for i := 0; i < len(path)-1; i++ {
			for _, e := range sub.EdgesFrom(path[i]) {
				if e.To == path[i+1] && e.Kind == graph.EdgeCalls {
					newG.AddEdge(&graph.Edge{From: e.From, To: e.To, Kind: e.Kind, Location: e.Location})
					break
				}
			}
		}
		g = newG
	}

	if depsFrom != "" {
		if g.Node(depsFrom) == nil {
			return fmt.Errorf("node %q not found in graph", depsFrom)
		}
		highlightIDs = []string{depsFrom}
		keep := make(map[string]bool)
		for _, id := range g.ReachableFrom(depsFrom, graph.EdgeCalls) {
			keep[id] = true
		}
		keep[depsFrom] = true
		g = g.Subgraph(keep)
		if verboseFlag {
			fmt.Fprintf(os.Stderr, "Subgraph: %d nodes, %d edges\n", g.NodeCount(), g.EdgeCount())
		}
	}

	if depsEdges != "" {
		var kinds []graph.EdgeKind
		valid := map[string]graph.EdgeKind{
			"requires": graph.EdgeRequires,
			"refers":   graph.EdgeRefers,
			"calls":    graph.EdgeCalls,
			"reads":    graph.EdgeReads,
		}
		for _, s := range strings.Split(depsEdges, ",") {
			s = strings.TrimSpace(strings.ToLower(s))
			if s == "" {
				continue
			}
			k, ok := valid[s]
			if !ok {
				return fmt.Errorf("invalid edge type %q (use: requires, refers, calls, reads)", s)
			}
			kinds = append(kinds, k)
		}
		if len(kinds) == 0 {
			return fmt.Errorf("--edges: specify at least one type (requires, refers, calls, reads)")
		}
		g = g.SubgraphByEdgeKinds(kinds)
		if verboseFlag {
			fmt.Fprintf(os.Stderr, "Filtered by edge types: %d nodes, %d edges\n", g.NodeCount(), g.EdgeCount())
		}
	}

	bytes, err := g.ToJSON()
	if err != nil {
		return err
	}
	fmt.Print(htmlViewer(string(bytes), highlightIDs, analysisMessage))
	return nil
}

func htmlViewer(jsonData string, highlightIDs []string, analysisMessage string) string {
	jsonSafe := strings.ReplaceAll(jsonData, "</script>", "<\\/script>")
	msgJS := "null"
	if analysisMessage != "" {
		s := strings.ReplaceAll(analysisMessage, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		s = strings.ReplaceAll(s, "<", "\\u003c")
		s = strings.ReplaceAll(s, ">", "\\u003e")
		msgJS = `"` + s + `"`
	}
	highlightJS := "[]"
	if len(highlightIDs) > 0 {
		highlightJS = "[" + strings.Join(func() []string {
			out := make([]string, len(highlightIDs))
			for i, id := range highlightIDs {
				s := strings.ReplaceAll(id, `\`, `\\`)
				s = strings.ReplaceAll(s, `"`, `\"`)
				out[i] = `"` + s + `"`
			}
			return out
		}(), ",") + "]"
	}
	legendEntry := ""
	if len(highlightIDs) > 0 {
		legendEntry = "\n    <span class=\"highlight\"></span> focal"
	}
	return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>ARIT – Dependency graph</title>
  <script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
  <style>
    body { font-family: sans-serif; margin: 0; padding: 8px; }
    #info { margin-bottom: 8px; color: #666; font-size: 14px; }
    #mynetwork { width: 100%; height: 90vh; border: 1px solid #ddd; }
    .legend { display: flex; gap: 16px; margin-top: 8px; flex-wrap: wrap; }
    .legend span { display: inline-block; width: 12px; height: 12px; border-radius: 2px; }
    .legend .ns { background: #e1f5fe; }
    .legend .fn { background: #e8f5e9; }
    .legend .var { background: #fff3e0; }
    .legend .highlight { background: #fff8e1; border: 2px solid #ff8f00; }
  </style>
</head>
<body>
  <div id="info">Loading graph...</div>
  <div id="mynetwork"></div>
  <div class="legend">
    <span class="ns"></span> namespace
    <span class="fn"></span> function
    <span class="var"></span> var` + legendEntry + `
  </div>
  <script type="application/json" id="graph-data">` + jsonSafe + `</script>
  <script>
    var highlightIds = ` + highlightJS + `;
    var analysisMessage = ` + msgJS + `;
  </script>
  <script>
    try {
      var data = JSON.parse(document.getElementById('graph-data').textContent);
      if (!data.nodes || !data.edges) throw new Error('Invalid graph data');
      var isHighlight = {};
      highlightIds.forEach(function(id) { isHighlight[id] = true; });
      var nodes = new vis.DataSet(data.nodes.map(function(n) {
        var color = n.kind === 'namespace' ? '#e1f5fe' : n.kind === 'function' ? '#e8f5e9' : '#fff3e0';
        var opts = { id: n.id, label: n.id, title: n.id + ' (' + n.kind + ')' + (isHighlight[n.id] ? ' [focal]' : ''), color: color };
        if (isHighlight[n.id]) {
          opts.borderWidth = 3;
          opts.borderWidthSelected = 4;
          opts.color = { background: '#fff8e1', border: '#ff8f00' };
        }
        return opts;
      }));
      var edgeCount = {};
      data.edges.forEach(function(e) {
        var k = e.from + '|' + e.to + '|' + e.kind;
        edgeCount[k] = (edgeCount[k] || 0) + 1;
      });
      var seenKey = {};
      var edgeList = data.edges.filter(function(e) {
        var k = e.from + '|' + e.to + '|' + e.kind;
        if (seenKey[k]) return false;
        seenKey[k] = true;
        return true;
      }).map(function(e) {
        var k = e.from + '|' + e.to + '|' + e.kind;
        var n = edgeCount[k];
        var label = n > 1 ? e.kind + ' (' + n + ')' : e.kind;
        return { from: e.from, to: e.to, label: label, arrows: 'to', title: label };
      });
      var edges = new vis.DataSet(edgeList);
      var container = document.getElementById('mynetwork');
      var netData = { nodes: nodes, edges: edges };
      var options = {
        nodes: {
          shape: 'box',
          font: { size: 12 },
          margin: 10,
          widthConstraint: { minimum: 80 }
        },
        edges: {
          font: { size: 10, align: 'horizontal', background: 'rgba(255,255,255,0.7)' },
          smooth: { type: 'continuous', roundness: 0.4 }
        },
        physics: {
          enabled: true,
          solver: 'forceAtlas2Based',
          forceAtlas2Based: {
            gravitationalConstant: -80,
            centralGravity: 0.008,
            springLength: 220,
            springConstant: 0.06,
            avoidOverlap: 1.2
          }
        }
      };
      var network = new vis.Network(container, netData, options);
      var info = 'Nodes: ' + data.nodes.length + ', Edges: ' + data.edges.length + ' — drag to pan, scroll to zoom';
      if (highlightIds.length > 0) info += ' (focal nodes in orange)';
      if (analysisMessage) info = analysisMessage + ' — ' + info;
      document.getElementById('info').textContent = info;
    } catch (e) {
      document.getElementById('info').textContent = 'Error: ' + e.message;
    }
  </script>
</body>
</html>
`
}
