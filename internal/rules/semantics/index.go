package semantics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

type FileSummary struct {
	Path      string
	Namespace string
	Requires  map[string]string
	MacroRefs map[string]string
	Functions map[string]FunctionSummary
	Macros    map[string]MacroDefinition
}

// MacroArity records the syntactic parameters of one macro arity. It is
// intentionally limited to declaration facts; it does not claim anything
// about the effects or contracts of future macro callers.
type MacroArity struct {
	Parameters []string
	Variadic   bool
}

// MacroDefinition is the safe project-level evidence currently available for
// a macro. Call sites and argument effects require a separate analysis pass.
type MacroDefinition struct {
	Name          string
	QualifiedName string
	Path          string
	Namespace     string
	Arities       []MacroArity
}

// MacroArgumentEvidence describes only facts that can be established from an
// argument form itself. Unknown calls remain unknown; no project convention is
// treated as an effect proof.
type MacroArgumentEvidence struct {
	Constant bool
	Effects  Effect
	Evidence Evidence
}

type MacroCallSite struct {
	Path             string
	Namespace        string
	Name             string
	QualifiedName    string
	Location         *reader.Location
	Arguments        []*reader.RichNode
	ArgumentEvidence []MacroArgumentEvidence
}

type ProjectIndex struct {
	Files              map[string]FileSummary
	Namespaces         map[string]string
	Definitions        map[string]string
	Macros             map[string]MacroDefinition
	MacroCalls         []MacroCallSite
	fileRoots          map[string][]*reader.RichNode
	fileComments       map[string][]*reader.RichNode
	indexErrors        map[string]string
	macroCallsComplete bool
}

func NewProjectIndex() *ProjectIndex {
	return &ProjectIndex{
		Files:        make(map[string]FileSummary),
		Namespaces:   make(map[string]string),
		Definitions:  make(map[string]string),
		Macros:       make(map[string]MacroDefinition),
		fileRoots:    make(map[string][]*reader.RichNode),
		fileComments: make(map[string][]*reader.RichNode),
		indexErrors:  make(map[string]string),
	}
}

func (index *ProjectIndex) AddFile(path, namespace string, roots []*reader.RichNode) {
	index.addFile(path, namespace, roots, nil)
}

func (index *ProjectIndex) addFile(path, namespace string, roots, comments []*reader.RichNode) {
	if index == nil {
		return
	}
	if index.Files == nil {
		index.Files = make(map[string]FileSummary)
	}
	if index.Namespaces == nil {
		index.Namespaces = make(map[string]string)
	}
	if index.Definitions == nil {
		index.Definitions = make(map[string]string)
	}
	if index.Macros == nil {
		index.Macros = make(map[string]MacroDefinition)
	}
	if index.fileRoots == nil {
		index.fileRoots = make(map[string][]*reader.RichNode)
	}
	if index.fileComments == nil {
		index.fileComments = make(map[string][]*reader.RichNode)
	}
	if index.indexErrors == nil {
		index.indexErrors = make(map[string]string)
	}

	macros := collectMacroDefinitions(path, namespace, roots)
	delete(index.indexErrors, path)
	index.macroCallsComplete = false
	index.fileRoots[path] = roots
	index.fileComments[path] = comments
	summary := FileSummary{
		Path:      path,
		Namespace: namespace,
		Requires:  collectRequires(roots),
		MacroRefs: collectMacroRefs(roots),
		Functions: BuildFunctionSummaries(roots, namespace),
		Macros:    macros,
	}
	index.Files[path] = summary
	if namespace != "" {
		index.Namespaces[namespace] = path
	}
	for name := range summary.Functions {
		if strings.Contains(name, "/") {
			index.Definitions[name] = path
		}
	}
	for name, macro := range macros {
		index.Macros[name] = macro
	}
}

// CachedFile returns the rich AST and comments prepared while indexing path.
// The returned nodes are intentionally shared with the index: cross-namespace
// analysis must not parse and rebuild the same file a second time.
func (index *ProjectIndex) CachedFile(path string) ([]*reader.RichNode, []*reader.RichNode, bool) {
	if index == nil || index.fileRoots == nil {
		return nil, nil, false
	}
	roots, found := index.fileRoots[path]
	if !found {
		return nil, nil, false
	}
	comments, commentsFound := index.fileComments[path]
	if !commentsFound || comments == nil {
		// AddFile callers may provide only an AST. Do not reuse it for analysis
		// because missing comments could change suppression behavior.
		return nil, nil, false
	}
	return roots, comments, true
}

func (index *ProjectIndex) MacroDefinitionOf(qualifiedName string) (MacroDefinition, bool) {
	if index == nil {
		return MacroDefinition{}, false
	}
	macro, ok := index.Macros[qualifiedName]
	return macro, ok
}

// BuildMacroCallSites performs the second pass needed after all project files
// have been indexed. It records only calls resolving to an indexed macro and
// ignores quoted, syntax-quoted, discarded, and macro-definition bodies. The
// caller must invoke MarkMacroCallSitesComplete after the complete file set has
// been indexed; until then, macro callers are not safe evidence for promotion.
func (index *ProjectIndex) BuildMacroCallSites() []MacroCallSite {
	if index == nil {
		return nil
	}
	index.MacroCalls = nil
	paths := make([]string, 0, len(index.fileRoots))
	for path := range index.fileRoots {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		roots := index.fileRoots[path]
		file := index.Files[path]
		var walk func(*reader.RichNode)
		walk = func(node *reader.RichNode) {
			if node == nil {
				return
			}
			switch node.Type {
			case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
				return
			}
			if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
				head := node.Children[0].Value
				if head == "defmacro" {
					return
				}
				if qualified, ok := index.resolveMacroCall(head, file.Namespace, file.Requires, file.MacroRefs); ok {
					arguments := append([]*reader.RichNode(nil), node.Children[1:]...)
					evidence := make([]MacroArgumentEvidence, 0, len(arguments))
					for _, argument := range arguments {
						evidence = append(evidence, summarizeMacroArgument(argument))
					}
					index.MacroCalls = append(index.MacroCalls, MacroCallSite{
						Path:             path,
						Namespace:        file.Namespace,
						Name:             head,
						QualifiedName:    qualified,
						Location:         node.Location,
						Arguments:        arguments,
						ArgumentEvidence: evidence,
					})
				}
			}
			for _, child := range node.Children {
				walk(child)
			}
		}
		for _, root := range roots {
			walk(root)
		}
	}
	sort.SliceStable(index.MacroCalls, func(i, j int) bool {
		left, right := index.MacroCalls[i], index.MacroCalls[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		leftLine, rightLine := 0, 0
		if left.Location != nil {
			leftLine = left.Location.StartLine
		}
		if right.Location != nil {
			rightLine = right.Location.StartLine
		}
		if leftLine != rightLine {
			return leftLine < rightLine
		}
		return left.QualifiedName < right.QualifiedName
	})
	index.macroCallsComplete = false
	return append([]MacroCallSite(nil), index.MacroCalls...)
}

// MarkMacroCallSitesComplete declares that indexing and caller discovery have
// covered the intended file set. An index error keeps the status incomplete.
func (index *ProjectIndex) MarkMacroCallSitesComplete() {
	if index == nil {
		return
	}
	index.macroCallsComplete = len(index.indexErrors) == 0
}

// MacroCallSitesComplete reports whether callers are complete for the file set
// explicitly supplied to this index. It does not claim that an undiscovered
// project file does not exist; the caller owns that coverage boundary.
func (index *ProjectIndex) MacroCallSitesComplete() bool {
	return index != nil && index.macroCallsComplete
}

// IndexErrors returns the paths that could not be indexed. The result is
// deterministic and detached from the index's internal state.
func (index *ProjectIndex) IndexErrors() []string {
	if index == nil {
		return nil
	}
	result := make([]string, 0, len(index.indexErrors))
	for path := range index.indexErrors {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func (index *ProjectIndex) MacroCallSites(qualifiedName string) []MacroCallSite {
	if index == nil {
		return nil
	}
	result := make([]MacroCallSite, 0)
	for _, call := range index.MacroCalls {
		if call.QualifiedName == qualifiedName {
			result = append(result, call)
		}
	}
	return result
}

func (index *ProjectIndex) resolveMacroCall(head, namespace string, requires, macroRefs map[string]string) (string, bool) {
	if slash := strings.Index(head, "/"); slash >= 0 {
		prefix, name := head[:slash], head[slash+1:]
		if resolved, ok := requires[prefix]; ok {
			qualified := resolved + "/" + name
			_, ok := index.Macros[qualified]
			return qualified, ok
		}
		_, ok := index.Macros[head]
		return head, ok
	}

	localQualified := ""
	if namespace != "" {
		localQualified = namespace + "/" + head
	}
	referredQualified := ""
	if referredNamespace, ok := macroRefs[head]; ok {
		referredQualified = referredNamespace + "/" + head
	}
	_, localExists := index.Macros[localQualified]
	_, referredExists := index.Macros[referredQualified]
	if localExists && referredExists && localQualified != referredQualified {
		return "", false
	}
	if referredExists {
		return referredQualified, true
	}
	return localQualified, localExists
}

func summarizeMacroArgument(node *reader.RichNode) MacroArgumentEvidence {
	evidence := MacroArgumentEvidence{Evidence: EvidenceUnknown}
	if node == nil {
		return evidence
	}
	constant, _ := constantValue(node)
	if constant {
		evidence.Constant = true
		evidence.Evidence = EvidenceProven
		return evidence
	}
	if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
		head := node.Children[0].Value
		facts := ForNode(node, nil)
		knownEffects := facts.Effects &^ EffectUnknown
		// An explicitly qualified core call has a stable semantic identity even
		// when the generic fact collector also marks unresolved child symbols.
		// Unqualified or external calls remain unknown to avoid shadowing FPs.
		if strings.HasPrefix(head, "clojure.core/") && knownEffects != EffectNone {
			evidence.Effects = knownEffects
			evidence.Evidence = EvidenceProven
			return evidence
		}
	}
	return evidence
}

func (index *ProjectIndex) HasNamespace(namespace string) bool {
	if index == nil {
		return false
	}
	_, ok := index.Namespaces[namespace]
	return ok
}

func (index *ProjectIndex) DefinitionOf(qualifiedName string) (string, bool) {
	if index == nil {
		return "", false
	}
	path, ok := index.Definitions[qualifiedName]
	return path, ok
}

func (index *ProjectIndex) EnrichResolutions(roots []*reader.RichNode) {
	if index == nil {
		return
	}
	for _, root := range roots {
		index.enrichNode(root)
	}
}

func (index *ProjectIndex) enrichNode(node *reader.RichNode) {
	if node == nil {
		return
	}
	if node.Resolution != nil && node.Resolution.Namespace != "" {
		node.Resolution.NamespaceKnown = index.HasNamespace(node.Resolution.Namespace) ||
			strings.HasPrefix(node.Resolution.Namespace, "clojure.") ||
			strings.HasPrefix(node.Resolution.Namespace, "cljs.") ||
			strings.HasPrefix(node.Resolution.Namespace, "java.")
	}
	for _, child := range node.Children {
		index.enrichNode(child)
	}
}

func (index *ProjectIndex) IndexFile(path string) error {
	tree, err := reader.ParseFile(path)
	if err != nil {
		if index != nil {
			if index.indexErrors == nil {
				index.indexErrors = make(map[string]string)
			}
			index.indexErrors[path] = err.Error()
			index.macroCallsComplete = false
		}
		return fmt.Errorf("parse %s: %w", path, err)
	}
	roots, comments := reader.BuildRichTree(tree)
	index.addFile(path, namespaceName(roots), roots, comments)
	return nil
}

func namespaceName(roots []*reader.RichNode) string {
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) < 2 || root.Children[0] == nil || root.Children[0].Value != "ns" {
			continue
		}
		if root.Children[1] != nil && root.Children[1].Type == reader.NodeSymbol {
			return root.Children[1].Value
		}
	}
	return ""
}

func collectRequires(roots []*reader.RichNode) map[string]string {
	requires := make(map[string]string)
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) < 2 || root.Children[0] == nil || root.Children[0].Value != "ns" {
			continue
		}
		for _, clause := range root.Children[2:] {
			if clause == nil || clause.Type != reader.NodeList || len(clause.Children) < 2 || clause.Children[0] == nil || (clause.Children[0].Value != ":require" && clause.Children[0].Value != ":require-macros") {
				continue
			}
			for _, spec := range clause.Children[1:] {
				if spec == nil {
					continue
				}
				if spec.Type == reader.NodeSymbol {
					requires[spec.Value] = spec.Value
					continue
				}
				if spec.Type != reader.NodeVector || len(spec.Children) == 0 || spec.Children[0] == nil || spec.Children[0].Type != reader.NodeSymbol {
					continue
				}
				namespace := spec.Children[0].Value
				requires[namespace] = namespace
				for i := 1; i+1 < len(spec.Children); i++ {
					if spec.Children[i].Type == reader.NodeKeyword && spec.Children[i].Value == ":as" && spec.Children[i+1].Type == reader.NodeSymbol {
						requires[spec.Children[i+1].Value] = namespace
					}
				}
			}
		}
	}
	return requires
}

func collectMacroRefs(roots []*reader.RichNode) map[string]string {
	refs := make(map[string]string)
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) < 2 || root.Children[0] == nil || root.Children[0].Value != "ns" {
			continue
		}
		for _, clause := range root.Children[2:] {
			if clause == nil || clause.Type != reader.NodeList || len(clause.Children) < 2 || clause.Children[0] == nil || (clause.Children[0].Value != ":require" && clause.Children[0].Value != ":require-macros") {
				continue
			}
			for _, spec := range clause.Children[1:] {
				if spec == nil || spec.Type != reader.NodeVector || len(spec.Children) < 2 || spec.Children[0] == nil || spec.Children[0].Type != reader.NodeSymbol {
					continue
				}
				namespace := spec.Children[0].Value
				for i := 1; i+1 < len(spec.Children); i++ {
					if spec.Children[i].Type != reader.NodeKeyword || (spec.Children[i].Value != ":refer" && spec.Children[i].Value != ":refer-macros") || spec.Children[i+1].Type != reader.NodeVector {
						continue
					}
					for _, referred := range spec.Children[i+1].Children {
						if referred != nil && referred.Type == reader.NodeSymbol {
							refs[referred.Value] = namespace
						}
					}
				}
			}
		}
	}
	return refs
}

func collectMacroDefinitions(path, namespace string, roots []*reader.RichNode) map[string]MacroDefinition {
	macros := make(map[string]MacroDefinition)
	var walk func(*reader.RichNode)
	walk = func(node *reader.RichNode) {
		if node == nil {
			return
		}
		if node.Type == reader.NodeList && len(node.Children) >= 3 &&
			node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol &&
			node.Children[0].Value == "defmacro" && node.Children[1] != nil &&
			node.Children[1].Type == reader.NodeSymbol {
			name := node.Children[1].Value
			definition := MacroDefinition{
				Name:          name,
				QualifiedName: name,
				Path:          path,
				Namespace:     namespace,
				Arities:       collectMacroArities(node),
			}
			if namespace != "" {
				definition.QualifiedName = namespace + "/" + name
			}
			macros[definition.QualifiedName] = definition
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, root := range roots {
		walk(root)
	}
	return macros
}

func collectMacroArities(node *reader.RichNode) []MacroArity {
	if node == nil || len(node.Children) < 3 {
		return nil
	}
	start := 2
	for start < len(node.Children) && (node.Children[start].Type == reader.NodeString || node.Children[start].Type == reader.NodeMap) {
		start++
	}
	if start >= len(node.Children) {
		return nil
	}
	if node.Children[start].Type == reader.NodeVector {
		return []MacroArity{macroArityFromParams(node.Children[start])}
	}
	result := make([]MacroArity, 0)
	for _, child := range node.Children[start:] {
		if child == nil || child.Type != reader.NodeList || len(child.Children) < 2 || child.Children[0].Type != reader.NodeVector {
			continue
		}
		result = append(result, macroArityFromParams(child.Children[0]))
	}
	return result
}

func macroArityFromParams(params *reader.RichNode) MacroArity {
	arity := MacroArity{}
	if params == nil || params.Type != reader.NodeVector {
		return arity
	}
	for _, child := range params.Children {
		if child == nil || child.Type != reader.NodeSymbol {
			continue
		}
		if child.Value == "&" {
			arity.Variadic = true
			continue
		}
		arity.Parameters = append(arity.Parameters, child.Value)
	}
	return arity
}
