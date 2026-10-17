package clojurespecific

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

// Catalog definition (clj-smells-catalog README, Misused Threading): a
// threading pipeline whose value changes fundamentally in kind at each step
// (map -> File -> String -> map). Threading is meant for homogeneous,
// sequential transformations.
//
// The heuristic classifies only steps whose result kind is known. A keyword
// lookup step (:root) is evidence that its input is a map. The pipeline is
// reported when it spans at least three kinds AND crosses a boundary kind
// (File, URI/URL, I/O stream, Java object construction) after
// having transformed ordinary data; a leading read chain is just the source:
// navigating a map into a string, a count and a predicate stays silent
// (ARIT doc 30 counterexample), while turning data into a File or URI and back
// is reported.

type threadKind struct {
	name     string
	boundary bool
}

var threadKindByFunction = map[string]threadKind{
	// strings
	"str": {"string", false}, "format": {"string", false}, "name": {"string", false}, "subs": {"string", false},
	"pr-str": {"string", false}, ".toString": {"string", false}, ".getAbsolutePath": {"string", false},
	".getPath": {"string", false}, ".getName": {"string", false}, ".getHost": {"string", false},
	".getFile": {"string", false}, ".getCanonicalPath": {"string", false},
	// keywords and symbols
	"keyword": {"keyword", false}, "symbol": {"symbol", false},
	// numbers
	"count": {"number", false}, "inc": {"number", false}, "dec": {"number", false},
	"parse-long": {"number", false}, "parse-double": {"number", false},
	"Integer/parseInt": {"number", false}, "Long/parseLong": {"number", false}, "Double/parseDouble": {"number", false},
	// sequences
	"map": {"seq", false}, "mapv": {"seq", false}, "filter": {"seq", false}, "filterv": {"seq", false},
	"remove": {"seq", false}, "mapcat": {"seq", false}, "keep": {"seq", false}, "take": {"seq", false},
	"drop": {"seq", false}, "sort": {"seq", false}, "sort-by": {"seq", false}, "distinct": {"seq", false},
	"seq": {"seq", false}, "vals": {"seq", false}, "keys": {"seq", false}, "reverse": {"seq", false},
	"partition": {"seq", false}, "line-seq": {"seq", false},
	// maps
	"assoc": {"map", false}, "dissoc": {"map", false}, "update": {"map", false}, "merge": {"map", false},
	"select-keys": {"map", false}, "assoc-in": {"map", false}, "update-in": {"map", false},
	// boundaries: files, URIs, I/O and parsing
	"clojure.java.io/file": {"file", true}, "java.io.File.": {"file", true}, "File.": {"file", true},
	"clojure.java.io/resource": {"url", true}, "clojure.java.io/as-url": {"url", true},
	"java.net.URI.": {"uri", true}, "URI.": {"uri", true}, "java.net.URL.": {"url", true}, "URL.": {"url", true},
	"clojure.java.io/reader": {"stream", true}, "clojure.java.io/writer": {"stream", true},
	"clojure.java.io/input-stream": {"stream", true}, "clojure.java.io/output-stream": {"stream", true},
	// Reading and parsing serialized data change the kind but are not object
	// boundaries: (-> req :body slurp (json/parse-string true)) is idiomatic.
	"slurp": {"string", false}, "read-string": {"data", false},
	"clojure.edn/read-string": {"data", false}, "cheshire.core/parse-string": {"data", false},
	"clojure.data.json/read-str": {"data", false}, "jsonista.core/read-value": {"data", false},
}

// Aliases left unresolved by the analyzer are matched by their usual names.
var threadKindAliasFallback = map[string]string{
	"io/file": "clojure.java.io/file", "io/resource": "clojure.java.io/resource", "io/as-url": "clojure.java.io/as-url",
	"io/reader": "clojure.java.io/reader", "io/writer": "clojure.java.io/writer",
	"io/input-stream": "clojure.java.io/input-stream", "io/output-stream": "clojure.java.io/output-stream",
	"edn/read-string": "clojure.edn/read-string", "json/parse-string": "cheshire.core/parse-string",
	"json/read-str": "clojure.data.json/read-str",
}

func threadStepFunctionName(node *reader.RichNode) string {
	if node == nil || node.Type != reader.NodeSymbol {
		return ""
	}
	name := node.Value
	if node.Resolution != nil && node.Resolution.Kind != reader.ResolutionUnresolved &&
		node.Resolution.Kind != reader.ResolutionLocal && node.Resolution.CanonicalName != "" {
		name = node.Resolution.CanonicalName
	}
	name = strings.TrimPrefix(name, "clojure.core/")
	if strings.HasPrefix(name, "clojure.string/") {
		return "str"
	}
	if canonical, ok := threadKindAliasFallback[name]; ok {
		return canonical
	}
	if strings.HasPrefix(node.Value, "str/") {
		return "str"
	}
	return name
}

// threadStepKind returns the kind produced by a step and whether a keyword
// lookup shows that the step's input is a map.
func threadStepKind(step *reader.RichNode) (kind threadKind, known bool, inputIsMap bool) {
	if step == nil {
		return threadKind{}, false, false
	}
	switch step.Type {
	case reader.NodeKeyword:
		return threadKind{}, false, true
	case reader.NodeSymbol:
		return threadKindOfName(threadStepFunctionName(step))
	case reader.NodeList:
		if len(step.Children) == 0 || step.Children[0] == nil {
			return threadKind{}, false, false
		}
		head := step.Children[0]
		if head.Type == reader.NodeKeyword {
			return threadKind{}, false, true
		}
		name := threadStepFunctionName(head)
		// (-> x (->> (io/file dir))) delegates to its single inner step.
		if (name == "->>" || name == "->") && len(step.Children) == 2 {
			return threadStepKind(step.Children[1])
		}
		return threadKindOfName(name)
	}
	return threadKind{}, false, false
}

func threadKindOfName(name string) (threadKind, bool, bool) {
	if name == "" {
		return threadKind{}, false, false
	}
	if kind, ok := threadKindByFunction[name]; ok {
		return kind, true, false
	}
	// Java object construction (Foo.) other than the known ones.
	if strings.HasSuffix(name, ".") && len(name) > 1 && name[0] >= 'A' && name[0] <= 'Z' {
		return threadKind{"object", true}, true, false
	}
	if strings.HasSuffix(name, "?") {
		return threadKind{"boolean", false}, true, false
	}
	return threadKind{}, false, false
}

func (r *MisusedThreadingRule) heterogeneousPipeline(node *reader.RichNode, context map[string]interface{}, filepath string) *rules.Finding {
	if node == nil || len(node.Children) < 4 {
		return nil
	}
	kinds := []string{}
	seen := map[string]bool{}
	boundary := ""
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			kinds = append(kinds, name)
		}
	}
	// A boundary only counts after the pipeline has worked on ordinary data
	// (a non-boundary kind or a map lookup). A read chain at the start, such as
	// (-> "config.edn" io/resource slurp edn/read-string), is the idiomatic
	// source of the data, not a crossing.
	// The object must also be converted again after the boundary ("and back",
	// as in the catalog example File -> str): locating and reading a file or
	// resource ((-> entry :path io/file slurp)) is not reported.
	transformedData := false
	convertedBack := false
	for _, step := range node.Children[2:] {
		kind, known, inputIsMap := threadStepKind(step)
		if inputIsMap {
			add("map")
			transformedData = true
		}
		if !known {
			continue
		}
		add(kind.name)
		if boundary != "" && !kind.boundary && !threadReadStep(step) {
			// Only a conversion back into a scalar/structural value counts;
			// processing a collection of files (.listFiles, filter) is natural.
			switch kind.name {
			case "string", "keyword", "symbol", "map", "number":
				convertedBack = true
			}
		}
		if kind.boundary {
			if transformedData && boundary == "" {
				boundary = kind.name
			}
			continue
		}
		transformedData = true
	}
	if len(kinds) < 3 || boundary == "" || !convertedBack {
		return nil
	}
	return rules.SetContextualFindingWithEvidence(&rules.Finding{
		RuleID: r.ID,
		Message: fmt.Sprintf(
			"Threading pipeline changes the kind of its value across steps (%s), crossing a %s boundary. Threading macros read best for homogeneous transformations; consider naming intermediate values with let.",
			strings.Join(kinds, " -> "), boundary),
		Filepath: filepath,
		Location: node.Location,
		Severity: rules.ContextualSeverity(context, r.Severity),
		Tags:     rules.ContextualTags(context),
	}, "Whether a heterogeneous pipeline hurts readability is a judgement; the AST shows only the kinds of the known steps.", "pipeline-readability", "value-kind-contract")
}

// threadReadStep reports steps that read or parse the located object.
func threadReadStep(step *reader.RichNode) bool {
	name := ""
	switch {
	case step == nil:
		return false
	case step.Type == reader.NodeSymbol:
		name = threadStepFunctionName(step)
	case step.Type == reader.NodeList && len(step.Children) > 0:
		name = threadStepFunctionName(step.Children[0])
	}
	switch name {
	case "slurp", "read-string", "clojure.edn/read-string", "cheshire.core/parse-string",
		"clojure.data.json/read-str", "jsonista.core/read-value", "line-seq", "file-seq":
		return true
	}
	return false
}
