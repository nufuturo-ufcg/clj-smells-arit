package analyzer

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/functional"
)

var EnableExperimentalMacroExpansion bool

type AnalysisResult struct {
	Findings        []rules.Finding
	RichRoots       []*reader.RichNode
	GlobalScope     *Scope
	Namespace       string
	Aliases         []NamespaceAlias
	ReferredSymbols []ReferredSymbol
}

type Scope struct {
	parent          *Scope
	symbols         map[string]*SymbolInfo
	aliases         map[string]*NamespaceAlias
	referredSymbols map[string]*ReferredSymbol

	lookupCache map[string]*SymbolInfo
	cacheValid  bool
	mu          sync.RWMutex
}

type SymbolType string

const (
	TypeFunction        SymbolType = "function"
	TypeVariable        SymbolType = "variable"
	TypeParameter       SymbolType = "parameter"
	TypeNamespace       SymbolType = "namespace"
	TypeReferred        SymbolType = "referred"
	TypeJava            SymbolType = "java_class"
	TypeUnknown         SymbolType = "unknown"
	TypeCoreFunction    SymbolType = "core-function"
	TypeCoreSpecialForm SymbolType = "core-special-form"
	TypeAliased         SymbolType = "aliased"
)

type SymbolInfo struct {
	Name            string
	Definition      *reader.RichNode
	Type            SymbolType
	IsPrivate       bool
	IsUsed          bool
	OriginNamespace string
	TypeHint        string
	InferredType    string
}

type NamespaceAlias struct {
	Alias          string
	FullNamespace  string
	DefinitionNode *reader.RichNode
}

type ReferredSymbol struct {
	SymbolName        string
	OriginalNamespace string
	DefinitionNode    *reader.RichNode
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:          parent,
		symbols:         make(map[string]*SymbolInfo),
		aliases:         make(map[string]*NamespaceAlias),
		referredSymbols: make(map[string]*ReferredSymbol),
		lookupCache:     make(map[string]*SymbolInfo),
		cacheValid:      true,
	}
}

func (s *Scope) Define(info *SymbolInfo) bool {
	if s == nil || info == nil {
		return false
	}

	if s.symbols == nil {
		s.symbols = make(map[string]*SymbolInfo)
	}

	if _, exists := s.symbols[info.Name]; exists {
		return false
	}
	s.symbols[info.Name] = info

	s.invalidateCache()
	return true
}

func (s *Scope) DefineAlias(alias NamespaceAlias) {
	if s == nil {
		return
	}

	if s.aliases == nil {
		s.aliases = make(map[string]*NamespaceAlias)
	}
	s.aliases[alias.Alias] = &alias
}

func (s *Scope) DefineReferredSymbol(ref ReferredSymbol) {
	if s == nil {
		return
	}

	if s.referredSymbols == nil {
		s.referredSymbols = make(map[string]*ReferredSymbol)
	}
	s.referredSymbols[ref.SymbolName] = &ref
}

func (s *Scope) invalidateCache() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cacheValid {
		return
	}

	s.cacheValid = false

	if s.lookupCache != nil {
		s.lookupCache = nil
	}

	if s.parent != nil && s.parent.cacheValid {
		go s.parent.invalidateCache()
	}
}

func (s *Scope) findLocalOrParentDef(name string) (*SymbolInfo, bool) {
	if s == nil || name == "" {
		return nil, false
	}

	s.mu.RLock()

	if s.cacheValid && s.lookupCache != nil {
		if info, found := s.lookupCache[name]; found {
			s.mu.RUnlock()
			return info, info != nil
		}
	}
	s.mu.RUnlock()

	current := s
	for current != nil {
		if current.symbols != nil {
			if info, found := current.symbols[name]; found && info != nil {

				s.mu.Lock()
				if s.lookupCache == nil && s.cacheValid {
					s.lookupCache = make(map[string]*SymbolInfo, 32)
				}
				if s.cacheValid && s.lookupCache != nil {
					s.lookupCache[name] = info
				}
				s.mu.Unlock()
				return info, true
			}
		}
		current = current.parent
	}

	s.mu.Lock()
	if s.cacheValid {
		if s.lookupCache == nil {
			s.lookupCache = make(map[string]*SymbolInfo, 32)
		}
		s.lookupCache[name] = nil
	}
	s.mu.Unlock()

	return nil, false
}

func (s *Scope) findAlias(aliasName string) (*NamespaceAlias, bool) {
	if s == nil || aliasName == "" {
		return nil, false
	}

	current := s
	for current != nil {
		if current.aliases != nil {
			if aliasInfo, found := current.aliases[aliasName]; found && aliasInfo != nil {
				return aliasInfo, true
			}
		}
		current = current.parent
	}
	return nil, false
}

func CollectDefinitions(nodes []*reader.RichNode, globalScope *Scope) {
	if globalScope == nil {
		return
	}

	localDefs := make(map[*reader.RichNode]*SymbolInfo)

	var visit func(node *reader.RichNode, currentScope *Scope)
	visit = func(node *reader.RichNode, currentScope *Scope) {
		if node == nil || currentScope == nil {
			return
		}

		nextScope := currentScope

		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && node.Children[0].Type == reader.NodeSymbol {
			funcNameNode := node.Children[0]
			switch funcNameNode.Value {
			case "defn", "defn-", "defmacro", "defmethod", "defmulti":
				if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeSymbol {
					funcSymbolNode := node.Children[1]
					var typeHint string
					if funcSymbolNode.TypeHint != "" {
						typeHint = funcSymbolNode.TypeHint
					} else if node.TypeHint != "" {
						typeHint = node.TypeHint
					}
					funcInfo := &SymbolInfo{
						Name:         funcSymbolNode.Value,
						Definition:   node,
						Type:         TypeFunction,
						IsPrivate:    funcNameNode.Value == "defn-",
						IsUsed:       false,
						TypeHint:     typeHint,
						InferredType: "Function",
					}
					currentScope.Define(funcInfo)

					fnScope := NewScope(currentScope)
					nextScope = fnScope

					paramIndex := 2
					if len(node.Children) > paramIndex && node.Children[paramIndex] != nil && node.Children[paramIndex].Type == reader.NodeString {
						paramIndex++
					}
					if len(node.Children) > paramIndex && node.Children[paramIndex] != nil && node.Children[paramIndex].Type == reader.NodeMap {
						paramIndex++
					}
					if len(node.Children) > paramIndex && node.Children[paramIndex] != nil {
						paramsNodeCandidate := node.Children[paramIndex]
						switch paramsNodeCandidate.Type {
						case reader.NodeVector:
							defineParams(paramsNodeCandidate, fnScope, localDefs)
						case reader.NodeList:
							for _, arityForm := range paramsNodeCandidate.Children {
								if arityForm != nil && arityForm.Type == reader.NodeList && len(arityForm.Children) > 0 && arityForm.Children[0] != nil && arityForm.Children[0].Type == reader.NodeVector {
									defineParams(arityForm.Children[0], fnScope, localDefs)
								}
							}
						}
					}
				}
			case "fn":
				fnScope := NewScope(currentScope)
				nextScope = fnScope
				paramIndex := 1

				if len(node.Children) > paramIndex && node.Children[paramIndex] != nil && node.Children[paramIndex].Type == reader.NodeSymbol {
					paramIndex++
				}

				if len(node.Children) > paramIndex && node.Children[paramIndex] != nil {
					paramsNodeCandidate := node.Children[paramIndex]
					switch paramsNodeCandidate.Type {
					case reader.NodeVector:
						defineParams(paramsNodeCandidate, fnScope, localDefs)
					case reader.NodeList:
						for _, arityForm := range paramsNodeCandidate.Children {
							if arityForm != nil && arityForm.Type == reader.NodeList && len(arityForm.Children) > 0 && arityForm.Children[0] != nil && arityForm.Children[0].Type == reader.NodeVector {
								defineParams(arityForm.Children[0], fnScope, localDefs)
							}
						}
					}
				}

			case "let", "loop":
				if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeVector {
					bindingsNode := node.Children[1]
					letScope := NewScope(currentScope)
					nextScope = letScope

					for i := 0; i < len(bindingsNode.Children); i += 2 {
						if i+1 >= len(bindingsNode.Children) {
							break
						}
						bindingVarNode := bindingsNode.Children[i]
						if bindingVarNode != nil {
							bindingValNode := bindingsNode.Children[i+1]
							defineBindingFormWithValue(bindingVarNode, bindingValNode, letScope, localDefs, TypeVariable)
						}
					}
				}
			case "def", "defonce":
				if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeSymbol {
					varSymbolNode := node.Children[1]
					var typeHint string
					if varSymbolNode.TypeHint != "" {
						typeHint = varSymbolNode.TypeHint
					} else if node.TypeHint != "" {
						typeHint = node.TypeHint
					}
					var inferredType string
					if len(node.Children) > 2 && node.Children[2] != nil {
						inferredType = node.Children[2].InferredType
					}
					varInfo := &SymbolInfo{
						Name:         varSymbolNode.Value,
						Definition:   node,
						Type:         TypeVariable,
						IsUsed:       false,
						TypeHint:     typeHint,
						InferredType: inferredType,
					}
					currentScope.Define(varInfo)
				}
			case "ns":
				return
			}
		}

		for idx, child := range node.Children {
			if child == nil {
				continue
			}

			currentChildScope := nextScope

			isLetLoopBindingVector := false
			if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && (node.Children[0].Value == "let" || node.Children[0].Value == "loop") {
				if idx == 1 && child.Type == reader.NodeVector {
					isLetLoopBindingVector = true
					for bindingValIdx := 1; bindingValIdx < len(child.Children); bindingValIdx += 2 {
						if bindingValIdx < len(child.Children) && child.Children[bindingValIdx] != nil {
							bindingValNode := child.Children[bindingValIdx]
							visit(bindingValNode, currentScope)
						}
					}
				} else if idx > 1 {
					currentChildScope = nextScope
				}
			}

			if isLetLoopBindingVector || shouldSkipChildInPass1(node, child, idx) {
				continue
			}

			visit(child, currentChildScope)
		}
	}

	for _, root := range nodes {
		if root != nil {
			visit(root, globalScope)
		}
	}
}

func ResolveSymbols(nodes []*reader.RichNode, globalScope *Scope) {
	var visit func(node *reader.RichNode, currentScope *Scope)
	visit = func(node *reader.RichNode, currentScope *Scope) {
		if node == nil {
			return
		}

		nextScope := currentScope

		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
			funcNameNodeVal := node.Children[0].Value
			switch funcNameNodeVal {
			case "defn", "defn-", "defmacro", "defmethod", "defmulti":
				if len(node.Children) > 1 && node.Children[1].Type == reader.NodeSymbol {

					newFnScope := NewScope(currentScope)
					paramIndex := 2
					if len(node.Children) > paramIndex && node.Children[paramIndex].Type == reader.NodeString {
						paramIndex++
					}
					if len(node.Children) > paramIndex && node.Children[paramIndex].Type == reader.NodeMap {
						paramIndex++
					}
					if len(node.Children) > paramIndex {
						paramsNode := node.Children[paramIndex]
						switch paramsNode.Type {
						case reader.NodeVector:
							defineParams(paramsNode, newFnScope, nil)
						case reader.NodeList:
							for _, arityForm := range paramsNode.Children {
								if arityForm.Type == reader.NodeList && len(arityForm.Children) > 0 && arityForm.Children[0].Type == reader.NodeVector {
									defineParams(arityForm.Children[0], newFnScope, nil)
								}
							}
						}
					}
					nextScope = newFnScope

				}
			case "fn":
				newFnScope := NewScope(currentScope)
				paramIndex := 1
				if len(node.Children) > paramIndex && node.Children[paramIndex].Type == reader.NodeSymbol {
					paramIndex++
				}
				if len(node.Children) > paramIndex {
					paramsNode := node.Children[paramIndex]
					switch paramsNode.Type {
					case reader.NodeVector:
						defineParams(paramsNode, newFnScope, nil)
					case reader.NodeList:
						for _, arityForm := range paramsNode.Children {
							if arityForm.Type == reader.NodeList && len(arityForm.Children) > 0 && arityForm.Children[0].Type == reader.NodeVector {
								defineParams(arityForm.Children[0], newFnScope, nil)
							}
						}
					}
				}
				nextScope = newFnScope

			case "let", "loop":
				if len(node.Children) > 1 && node.Children[1].Type == reader.NodeVector {
					newLetScope := NewScope(currentScope)
					bindingsNode := node.Children[1]
					for i := 0; i < len(bindingsNode.Children); i += 2 {
						if i+1 < len(bindingsNode.Children) {
							bindingVarNode := bindingsNode.Children[i]
							bindingValNode := bindingsNode.Children[i+1]
							defineBindingFormWithValue(bindingVarNode, bindingValNode, newLetScope, nil, TypeVariable)
						}
					}
					nextScope = newLetScope
				}

			}
		}

		if node.Type == reader.NodeSymbol {
			symbolName := node.Value
			if info, found := currentScope.findLocalOrParentDef(symbolName); found {
				node.ResolvedDefinition = info.Definition
				node.SymbolRef = info
				info.IsUsed = true
				if info.TypeHint != "" {
					node.TypeHint = info.TypeHint
				}

			} else if aliasInfo, aliasFound := currentScope.findAlias(symbolName); aliasFound {
				node.SymbolRef = aliasInfo

			} else {

			}
			node.Resolution = resolveSymbolIdentity(node, currentScope)
		}

		for idx, child := range node.Children {
			currentChildScope := nextScope

			if node.Type == reader.NodeList && len(node.Children) > 0 && (node.Children[0].Value == "let" || node.Children[0].Value == "loop") {
				if idx == 1 && child.Type == reader.NodeVector {

					for bindingValIdx := 1; bindingValIdx < len(child.Children); bindingValIdx += 2 {
						bindingValNode := child.Children[bindingValIdx]
						visit(bindingValNode, currentScope)
					}
					continue
				} else if idx > 1 {
					currentChildScope = nextScope
				}
			}

			if shouldSkipChildInPass1(node, child, idx) {
				continue
			}
			visit(child, currentChildScope)
		}
	}

	for _, rootNode := range nodes {
		visit(rootNode, globalScope)
	}
}

func javaClassName(name string, scope *Scope) (string, bool) {
	if name == "" {
		return "", false
	}
	if info, found := scope.Lookup(name); found && info != nil && info.Type == TypeJava {
		return info.OriginNamespace, true
	}
	lastSegment := name
	if dot := strings.LastIndex(lastSegment, "."); dot >= 0 {
		lastSegment = lastSegment[dot+1:]
	}
	if lastSegment != "" && lastSegment[0] >= 'A' && lastSegment[0] <= 'Z' {
		return name, true
	}
	return "", false
}

func resolveSymbolIdentity(node *reader.RichNode, scope *Scope) *reader.SymbolResolution {
	symbol := ""
	if node != nil {
		symbol = node.Value
	}
	resolution := &reader.SymbolResolution{
		Kind: reader.ResolutionUnresolved, CanonicalName: symbol, Name: symbol,
	}
	if symbol == "" {
		return resolution
	}
	if strings.HasPrefix(symbol, ".") {
		resolution.Kind = reader.ResolutionJavaMethod
		resolution.Name = strings.TrimPrefix(symbol, ".")
		return resolution
	}
	if strings.HasSuffix(symbol, ".") {
		classSymbol := strings.TrimSuffix(symbol, ".")
		if className, ok := javaClassName(classSymbol, scope); ok {
			resolution.Kind = reader.ResolutionJavaConstructor
			resolution.CanonicalName = className + "."
			resolution.Namespace = className
			resolution.Name = classSymbol
		}
		return resolution
	}

	if slash := strings.Index(symbol, "/"); slash >= 0 {
		prefix, member := symbol[:slash], symbol[slash+1:]
		resolution.Name = member
		if prefix == "clojure.core" {
			resolution.Kind = reader.ResolutionClojureCore
			resolution.CanonicalName = "clojure.core/" + member
			resolution.Namespace = "clojure.core"
			return resolution
		}
		if className, ok := javaClassName(prefix, scope); ok {
			resolution.Kind = reader.ResolutionJavaStatic
			resolution.CanonicalName = className + "/" + member
			resolution.Namespace = className
			return resolution
		}
		if alias, ok := scope.findAlias(prefix); ok && alias != nil {
			resolution.Kind = reader.ResolutionNamespaceVar
			resolution.CanonicalName = alias.FullNamespace + "/" + member
			resolution.Namespace = alias.FullNamespace
			return resolution
		}
		// A fully-qualified Clojure var needs no alias to be exact. Java
		// classes have already been separated above by their class segment.
		if strings.Contains(prefix, ".") {
			resolution.Kind = reader.ResolutionNamespaceVar
			resolution.CanonicalName = symbol
			resolution.Namespace = prefix
		}
		return resolution
	}
	if className, ok := javaClassName(symbol, scope); ok {
		resolution.Kind = reader.ResolutionJavaStatic
		resolution.CanonicalName = className
		resolution.Namespace = className
		return resolution
	}

	if info, found := scope.Lookup(symbol); found && info != nil {
		// Definition collection is intentionally a whole-file pass. Preserve
		// Clojure's compilation order when a later top-level definition happens
		// to shadow a clojure.core symbol used earlier in the file.
		if info.Definition != nil && info.Definition.Location != nil && node.Location != nil &&
			info.Definition.Location.StartLine > node.Location.StartLine {
			if _, core := coreSymbols[symbol]; core {
				resolution.Kind = reader.ResolutionClojureCore
				resolution.CanonicalName = "clojure.core/" + symbol
				resolution.Namespace = "clojure.core"
				return resolution
			}
		}
		resolution.Name = symbol
		switch info.Type {
		case TypeCoreFunction, TypeCoreSpecialForm:
			resolution.Kind = reader.ResolutionClojureCore
			resolution.CanonicalName = "clojure.core/" + symbol
			resolution.Namespace = "clojure.core"
		case TypeReferred:
			if strings.HasPrefix(info.OriginNamespace, "clojure.") || !strings.Contains(info.OriginNamespace, ".") {
				resolution.Kind = reader.ResolutionNamespaceVar
				resolution.CanonicalName = info.OriginNamespace + "/" + symbol
				resolution.Namespace = info.OriginNamespace
			} else {
				resolution.Kind = reader.ResolutionLocal
			}
		case TypeJava:
			resolution.Kind = reader.ResolutionJavaStatic
			resolution.CanonicalName = info.OriginNamespace
			resolution.Namespace = info.OriginNamespace
		default:
			resolution.Kind = reader.ResolutionLocal
		}
	}
	return resolution
}

type Analyzer struct {
	Rules  []rules.CheckerRule
	Config *config.Config
}

func NewAnalyzer(cfg *config.Config) *Analyzer {
	analyzer := &Analyzer{
		Config: cfg,
	}

	allRuleInstances := rules.AllRules()

	for _, ruleInstance := range allRuleInstances {
		checkerRule, ok := ruleInstance.(rules.CheckerRule)
		if !ok {

			continue
		}

		ruleMetaID := checkerRule.Meta().ID
		ruleGroup := rules.GetRuleGroup(ruleMetaID)

		groupEnabled, groupSpecified := cfg.EnabledGroups[ruleGroup]
		ruleEnabled, ruleSpecified := cfg.EnabledRules[ruleMetaID]

		var shouldProcessRule bool
		if groupSpecified {
			shouldProcessRule = groupEnabled
		} else if ruleSpecified {
			shouldProcessRule = ruleEnabled
		} else {
			// By default, if nothing is specified in config, only clojure-specific is enabled
			shouldProcessRule = (ruleGroup == "clojure-specific")
		}

		if !shouldProcessRule {
			continue
		}

		clonedRule := cloneRule(checkerRule)
		configuredRule := configureRule(clonedRule, cfg)

		analyzer.Rules = append(analyzer.Rules, configuredRule)
	}

	return analyzer
}

func cloneRule(rule rules.CheckerRule) rules.CheckerRule {
	val := reflect.ValueOf(rule)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	newVal := reflect.New(val.Type())
	newVal.Elem().Set(val)

	return newVal.Interface().(rules.CheckerRule)
}

func configureRule(rule rules.CheckerRule, cfg *config.Config) rules.CheckerRule {
	ruleMetaID := rule.Meta().ID
	ruleCfg, cfgExists := cfg.RuleConfig[ruleMetaID]

	if !cfgExists {
		return rule
	}

	if typedRule, ok := rule.(*functional.LazySideEffectsRule); ok {
		newRule := &functional.LazySideEffectsRule{
			LazyContextFuncs: make(map[string]bool),
			SideEffectFuncs:  make(map[string]bool),
		}

		for k, v := range functional.DefaultLazyContextFunctions {
			newRule.LazyContextFuncs[k] = v
		}
		for k, v := range functional.DefaultSideEffectFunctions {
			newRule.SideEffectFuncs[k] = v
		}

		if funcs, ok := ruleCfg["lazy_context_funcs"].(map[string]interface{}); ok {
			for k, v := range funcs {
				if enabled, okBool := v.(bool); okBool {
					newRule.LazyContextFuncs[k] = enabled
				}
			}
		}
		if funcs, ok := ruleCfg["side_effect_funcs"].(map[string]interface{}); ok {
			for k, v := range funcs {
				if enabled, okBool := v.(bool); okBool {
					newRule.SideEffectFuncs[k] = enabled
				}
			}
		}

		_ = typedRule
		return newRule
	}
	return rule
}

func isNodeEagerConsumer(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return false
	}
	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return false
	}
	_, isEager := functional.EagerConsumerFunctions[funcNode.Value]
	return isEager
}

var executionKnownEagerHeads = map[string]struct{}{
	"def": {}, "defonce": {},
	"let": {}, "let*": {}, "loop": {}, "loop*": {}, "binding": {},
	"do": {}, "if": {}, "if-let": {}, "if-not": {}, "when": {}, "when-let": {}, "when-not": {},
	"cond": {}, "condp": {}, "case": {}, "try": {}, "catch": {}, "finally": {},
	"->": {}, "->>": {}, "some->": {}, "some->>": {}, "cond->": {}, "cond->>": {}, "as->": {},
	"and": {}, "or": {}, "doto": {}, "..": {},
	"vector": {}, "hash-map": {}, "array-map": {}, "list": {}, "set": {}, "str": {},
	"assoc": {}, "dissoc": {}, "merge": {}, "conj": {}, "into": {},
}

func executionHead(node *reader.RichNode) string {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0].Type != reader.NodeSymbol {
		return ""
	}
	return node.Children[0].Value
}

func isLocallyDefinedMacro(head *reader.RichNode) bool {
	if head == nil || head.ResolvedDefinition == nil {
		return false
	}
	definition := head.ResolvedDefinition
	return definition.Type == reader.NodeList && len(definition.Children) > 0 &&
		definition.Children[0].Type == reader.NodeSymbol && definition.Children[0].Value == "defmacro"
}

func childExecutionContext(parent *reader.RichNode, childIndex int, inherited rules.ExecutionContext) rules.ExecutionContext {
	if inherited != rules.ExecutionAtLoad || parent == nil {
		return inherited
	}

	switch parent.Type {
	case reader.NodeQuote, reader.NodeSyntaxQuote, reader.NodeVarQuote, reader.NodeReaderDiscard:
		return rules.ExecutionNonEvaluated
	}

	head := executionHead(parent)
	if head == "" {
		return inherited
	}
	if childIndex == 0 {
		return inherited
	}

	switch head {
	case "comment", "quote", "clojure.core/quote":
		return rules.ExecutionNonEvaluated
	case "defn", "defn-", "defmacro", "fn", "fn*", "letfn", "deftest":
		return rules.ExecutionDeferred
	case "defmethod":
		// The dispatch value is evaluated while defining the method, but its
		// implementation body is invoked later.
		if childIndex >= 3 {
			return rules.ExecutionDeferred
		}
		return inherited
	case "delay", "lazy-seq", "future", "future-call":
		return rules.ExecutionDeferred
	}

	if _, known := executionKnownEagerHeads[head]; known {
		return inherited
	}

	// A locally defined macro is known to be a macro, but without expansion we
	// cannot prove when any of its arguments execute.
	if isLocallyDefinedMacro(parent.Children[0]) {
		return rules.ExecutionUnknown
	}

	// An unresolved call may be a macro supplied by a dependency. Ordinary
	// functions eagerly evaluate arguments, but guessing that distinction is
	// precisely what creates load-time false positives. Prefer silence.
	return rules.ExecutionUnknown
}

func (a *Analyzer) Analyze(filepath string, richRootNodes []*reader.RichNode, comments []*reader.RichNode, globalScope *Scope) []*rules.Finding {

	var findingsMutex sync.Mutex
	allFindings := []*rules.Finding{}

	var traverseAndAnalyze func(node *reader.RichNode, currentContext map[string]interface{}, scope *Scope)
	traverseAndAnalyze = func(node *reader.RichNode, currentContext map[string]interface{}, scope *Scope) {
		if node == nil {
			return
		}

		prevScope := currentContext["scope"]
		prevConfig := currentContext["config"]
		currentContext["scope"] = scope
		currentContext["config"] = a.Config

		for _, rule := range a.Rules {
			if finding := rule.Check(node, currentContext, filepath); finding != nil {
				// Inject fingerprint centrally: covers both DSL rules (via builder)
				// and hand-written detectors that instantiate Finding directly.
				if finding.ASTFingerprint == "" {
					finding.ASTFingerprint = rules.ComputeFingerprint(node)
				}
				findingsMutex.Lock()
				allFindings = append(allFindings, finding)
				findingsMutex.Unlock()
			}
		}

		prevParent := currentContext["parent"]
		currentContext["parent"] = node
		prevAncestors := currentContext["ancestorNodes"]
		if ancestors, ok := prevAncestors.([]*reader.RichNode); ok {
			currentContext["ancestorNodes"] = append(ancestors, node)
		}

		prevIsInEager, _ := currentContext["isInEagerContext"].(bool)
		nodeIsEager := isNodeEagerConsumer(node)
		eagerChanged := false
		if nodeIsEager && !prevIsInEager {
			currentContext["isInEagerContext"] = true
			eagerChanged = true
		}

		var prevEnclosing interface{}
		enclosingChanged := false
		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
			if enclosing, ok := currentContext["enclosingForms"].([]string); ok {
				prevEnclosing = enclosing
				currentContext["enclosingForms"] = append(enclosing, node.Children[0].Value)
				enclosingChanged = true
			}
		} else if node.Type == reader.NodeQuote || node.Type == reader.NodeVarQuote ||
			node.Type == reader.NodeReaderDiscard {
			if enclosing, ok := currentContext["enclosingForms"].([]string); ok {
				prevEnclosing = enclosing
				currentContext["enclosingForms"] = append(enclosing, "__non-evaluated__")
				enclosingChanged = true
			}
		}

		parentIsInsideFunc, _ := currentContext["isInsideFunction"].(bool)
		parentIsInsideLet, _ := currentContext["isInsideLet"].(bool)
		parentIsInsideLoop, _ := currentContext["isInsideLoop"].(bool)
		parentIsInsideBinding, _ := currentContext["isInsideBinding"].(bool)
		parentIsInsideDosync, _ := currentContext["isInsideDosync"].(bool)
		parentIsInsideWithOpen, _ := currentContext["isInsideWithOpen"].(bool)

		currentNodeDefinesFunc := false
		currentNodeDefinesLet := false
		currentNodeDefinesLoop := false
		currentNodeDefinesBinding := false
		currentNodeDefinesDosync := false
		currentNodeDefinesWithOpen := false

		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
			nodeVal := node.Children[0].Value
			switch nodeVal {
			case "defn", "defn-", "defmacro", "defmethod", "defmulti", "fn":
				currentNodeDefinesFunc = true
			case "let":
				currentNodeDefinesLet = true
			case "loop":
				currentNodeDefinesLoop = true
			case "binding":
				currentNodeDefinesBinding = true
			case "dosync":
				currentNodeDefinesDosync = true
			case "with-open":
				currentNodeDefinesWithOpen = true
			}
		}

		for idx, child := range node.Children {
			currentChildScope := scope

			if node.Type == reader.NodeList && len(node.Children) > 0 && (node.Children[0].Value == "let" || node.Children[0].Value == "loop") {
				if idx == 1 && child.Type == reader.NodeVector {
					currentChildScope = scope
				}
			}

			childIsInsideFunc := parentIsInsideFunc
			funcBodyStartIndex := -1
			if currentNodeDefinesFunc {
				funcBodyStartIndex = 2
				if node.Children[0].Value == "fn" {
					funcBodyStartIndex = 1
				}

				if len(node.Children) > funcBodyStartIndex && node.Children[funcBodyStartIndex].Type == reader.NodeSymbol {
					if node.Children[0].Value != "fn" || idx > funcBodyStartIndex {
						funcBodyStartIndex++
					}
				}
				if node.Children[0].Value != "fn" {
					if len(node.Children) > funcBodyStartIndex && node.Children[funcBodyStartIndex].Type == reader.NodeString {
						funcBodyStartIndex++
					}
					if len(node.Children) > funcBodyStartIndex && node.Children[funcBodyStartIndex].Type == reader.NodeMap {
						funcBodyStartIndex++
					}
				}

				if len(node.Children) > funcBodyStartIndex &&
					(node.Children[funcBodyStartIndex].Type == reader.NodeVector || node.Children[funcBodyStartIndex].Type == reader.NodeList) {
					funcBodyStartIndex++
				}

				if idx >= funcBodyStartIndex {
					childIsInsideFunc = true
				}
			}

			childIsInsideLet := parentIsInsideLet || (currentNodeDefinesLet && idx > 0)
			childIsInsideLoop := parentIsInsideLoop || (currentNodeDefinesLoop && idx > 0)
			childIsInsideBinding := parentIsInsideBinding || (currentNodeDefinesBinding && idx > 0)
			childIsInsideDosync := parentIsInsideDosync || (currentNodeDefinesDosync && idx > 0)
			childIsInsideWithOpen := parentIsInsideWithOpen || (currentNodeDefinesWithOpen && idx > 0)

			prevChildFunc := currentContext["isInsideFunction"]
			currentContext["isInsideFunction"] = childIsInsideFunc

			prevChildLet := currentContext["isInsideLet"]
			currentContext["isInsideLet"] = childIsInsideLet

			prevChildLoop := currentContext["isInsideLoop"]
			currentContext["isInsideLoop"] = childIsInsideLoop

			prevChildBinding := currentContext["isInsideBinding"]
			currentContext["isInsideBinding"] = childIsInsideBinding

			prevChildDosync := currentContext["isInsideDosync"]
			currentContext["isInsideDosync"] = childIsInsideDosync

			prevChildWithOpen := currentContext["isInsideWithOpen"]
			currentContext["isInsideWithOpen"] = childIsInsideWithOpen

			prevExecution := currentContext["executionContext"]
			inheritedExecution, _ := prevExecution.(rules.ExecutionContext)
			currentContext["executionContext"] = childExecutionContext(node, idx, inheritedExecution)

			traverseAndAnalyze(child, currentContext, currentChildScope)

			currentContext["isInsideFunction"] = prevChildFunc
			currentContext["isInsideLet"] = prevChildLet
			currentContext["isInsideLoop"] = prevChildLoop
			currentContext["isInsideBinding"] = prevChildBinding
			currentContext["isInsideDosync"] = prevChildDosync
			currentContext["isInsideWithOpen"] = prevChildWithOpen
			currentContext["executionContext"] = prevExecution
		}

		currentContext["parent"] = prevParent
		currentContext["ancestorNodes"] = prevAncestors
		if eagerChanged {
			currentContext["isInEagerContext"] = prevIsInEager
		}
		if enclosingChanged {
			currentContext["enclosingForms"] = prevEnclosing
		}
		currentContext["scope"] = prevScope
		currentContext["config"] = prevConfig
	}

	initialContext := map[string]interface{}{
		"isInEagerContext": false,
		"isInsideFunction": false,
		"isInsideLet":      false,
		"isInsideLoop":     false,
		"isInsideBinding":  false,
		"isInsideDosync":   false,
		"isInsideWithOpen": false,
		"executionContext": rules.ExecutionAtLoad,
		"enclosingForms":   make([]string, 0, 32),
		"ancestorNodes":    make([]*reader.RichNode, 0, 32),
		"namespace-aliases": func() map[string]string {
			aliases := make(map[string]string)
			if globalScope != nil {
				for alias, info := range globalScope.aliases {
					if info != nil {
						aliases[alias] = info.FullNamespace
					}
				}
			}
			return aliases
		}(),
	}

	for _, rootNode := range richRootNodes {
		traverseAndAnalyze(rootNode, initialContext, globalScope)
	}

	initialContext["scope"] = globalScope
	initialContext["config"] = a.Config

	for _, commentNode := range comments {
		for _, rule := range a.Rules {
			if finding := rule.Check(commentNode, initialContext, filepath); finding != nil {
				findingsMutex.Lock()
				allFindings = append(allFindings, finding)
				findingsMutex.Unlock()
			}
		}
	}

	delete(initialContext, "scope")
	delete(initialContext, "config")

	return allFindings
}

func defineParams(paramsNode *reader.RichNode, targetScope *Scope, localDefs map[*reader.RichNode]*SymbolInfo) {
	if paramsNode == nil || paramsNode.Type != reader.NodeVector {
		return
	}
	defineBindingForm(paramsNode, targetScope, localDefs, TypeParameter)
}

func defineBindingFormWithValue(bindingNode *reader.RichNode, valNode *reader.RichNode, targetScope *Scope, localDefs map[*reader.RichNode]*SymbolInfo, defaultSymbolType SymbolType) {
	if bindingNode == nil {
		return
	}
	if bindingNode.Type == reader.NodeSymbol {
		symbolName := bindingNode.Value
		if symbolName == "_" || symbolName == "&" || strings.HasPrefix(symbolName, ".") || strings.Contains(symbolName, "/") {
			return
		}
		info := &SymbolInfo{
			Name:       symbolName,
			Definition: bindingNode,
			Type:       defaultSymbolType,
			IsUsed:     false,
		}
		if bindingNode.TypeHint != "" {
			info.TypeHint = bindingNode.TypeHint
		}
		if valNode != nil {
			if info.TypeHint == "" && valNode.TypeHint != "" {
				info.TypeHint = valNode.TypeHint
			}
			info.InferredType = valNode.InferredType
		} else {
			info.InferredType = bindingNode.InferredType
		}
		if targetScope.Define(info) {
			if localDefs != nil {
				localDefs[bindingNode] = info
			}
		}
		return
	}
	defineBindingForm(bindingNode, targetScope, localDefs, defaultSymbolType)
}

func defineBindingForm(bindingNode *reader.RichNode, targetScope *Scope, localDefs map[*reader.RichNode]*SymbolInfo, defaultSymbolType SymbolType) {
	if bindingNode == nil {
		return
	}

	switch bindingNode.Type {
	case reader.NodeSymbol:
		symbolName := bindingNode.Value
		if symbolName == "_" || symbolName == "&" || strings.HasPrefix(symbolName, ".") || strings.Contains(symbolName, "/") {
			return
		}
		info := &SymbolInfo{
			Name:         symbolName,
			Definition:   bindingNode,
			Type:         defaultSymbolType,
			IsUsed:       false,
			TypeHint:     bindingNode.TypeHint,
			InferredType: bindingNode.InferredType,
		}
		if targetScope.Define(info) {
			if localDefs != nil {
				localDefs[bindingNode] = info
			}
		}

	case reader.NodeVector:
		for _, elem := range bindingNode.Children {
			defineBindingForm(elem, targetScope, localDefs, defaultSymbolType)
		}

	case reader.NodeMap:
		var asSymbolNode *reader.RichNode
		keysToDefine := []*reader.RichNode{}

		for i := 0; i < len(bindingNode.Children); i += 2 {
			keyNode := bindingNode.Children[i]
			if i+1 >= len(bindingNode.Children) {
				break
			}
			valueNode := bindingNode.Children[i+1]

			if keyNode.Type == reader.NodeKeyword {
				switch keyNode.Value {
				case "keys", "strs", "syms":
					if valueNode.Type == reader.NodeVector {
						for _, symInVec := range valueNode.Children {
							if symInVec.Type == reader.NodeSymbol {
								keysToDefine = append(keysToDefine, symInVec)
							}
						}
					}
				case "as":
					if valueNode.Type == reader.NodeSymbol {
						asSymbolNode = valueNode
					}

				}
			} else if valueNode.Type == reader.NodeSymbol {
				keysToDefine = append(keysToDefine, valueNode)
			}
		}
		for _, symToDef := range keysToDefine {
			defineBindingForm(symToDef, targetScope, localDefs, defaultSymbolType)
		}
		if asSymbolNode != nil {
			defineBindingForm(asSymbolNode, targetScope, localDefs, defaultSymbolType)
		}
	}
}

func shouldSkipChildInPass1(parentNode, childNode *reader.RichNode, childIndex int) bool {
	if parentNode.Type == reader.NodeList && len(parentNode.Children) > 0 && parentNode.Children[0].Type == reader.NodeSymbol {
		funcName := parentNode.Children[0].Value
		switch funcName {
		case "defn", "defn-", "defmacro", "defmethod", "defmulti":

			if childIndex == 1 {
				return true
			}

			paramDefIndex := 2
			if len(parentNode.Children) > paramDefIndex && parentNode.Children[paramDefIndex].Type == reader.NodeString {
				paramDefIndex++
			}
			if len(parentNode.Children) > paramDefIndex && parentNode.Children[paramDefIndex].Type == reader.NodeMap {
				paramDefIndex++
			}

			if childIndex < paramDefIndex {
				return true
			}
			if childIndex == paramDefIndex {
				return true
			}

		case "fn":

			paramDefIndex := 1
			if len(parentNode.Children) > paramDefIndex && parentNode.Children[paramDefIndex].Type == reader.NodeSymbol {
				if childIndex == paramDefIndex {
					return true
				}
				paramDefIndex++
			}
			if childIndex == paramDefIndex {
				return true
			}

		case "let", "loop":
			if childIndex == 1 && childNode.Type == reader.NodeVector {
				return true
			}
		case "def", "defonce":

			if childIndex == 1 {
				return true
			}
			if childIndex == 2 && childNode.Type == reader.NodeString {
				return true
			}
		case "ns":
			return true
		}
	}
	return false
}

func parseNamespaceForm(nsNode *reader.RichNode) (string, []NamespaceAlias, []ReferredSymbol, error) {
	if nsNode == nil || nsNode.Type != reader.NodeList || len(nsNode.Children) == 0 || nsNode.Children[0].Value != "ns" {
		return "", nil, nil, fmt.Errorf("node is not a valid ns form")
	}

	var namespaceName string
	var aliases []NamespaceAlias
	var referredSymbols []ReferredSymbol

	if len(nsNode.Children) > 1 && nsNode.Children[1].Type == reader.NodeSymbol {
		namespaceName = nsNode.Children[1].Value
	}

	for i := 2; i < len(nsNode.Children); i++ {
		clauseNode := nsNode.Children[i]
		if clauseNode.Type != reader.NodeList || len(clauseNode.Children) == 0 || clauseNode.Children[0].Type != reader.NodeKeyword {
			continue
		}
		clauseKeyword := strings.TrimPrefix(clauseNode.Children[0].Value, ":")

		switch clauseKeyword {
		case "require":
			for j := 1; j < len(clauseNode.Children); j++ {
				specNode := clauseNode.Children[j]
				if specNode.Type != reader.NodeVector || len(specNode.Children) == 0 {
					continue
				}
				nsToRequireNode := specNode.Children[0]
				if nsToRequireNode.Type != reader.NodeSymbol {
					continue
				}
				fullNs := nsToRequireNode.Value
				var currentAlias string
				var refers []string

				for k := 1; k < len(specNode.Children); k++ {
					optionKeyNode := specNode.Children[k]
					if optionKeyNode.Type != reader.NodeKeyword {
						continue
					}
					optionKey := strings.TrimPrefix(optionKeyNode.Value, ":")
					k++
					if k >= len(specNode.Children) {
						break
					}
					optionValueNode := specNode.Children[k]

					switch optionKey {
					case "as":
						if optionValueNode.Type == reader.NodeSymbol {
							currentAlias = optionValueNode.Value
						}
					case "refer":
						if optionValueNode.Type == reader.NodeVector {
							for _, referSymNode := range optionValueNode.Children {
								if referSymNode.Type == reader.NodeSymbol {
									refers = append(refers, referSymNode.Value)
								}
							}
						}
					}
				}
				if currentAlias != "" {
					aliases = append(aliases, NamespaceAlias{Alias: currentAlias, FullNamespace: fullNs, DefinitionNode: specNode})
				}
				for _, referSym := range refers {
					referredSymbols = append(referredSymbols, ReferredSymbol{SymbolName: referSym, OriginalNamespace: fullNs, DefinitionNode: specNode})
				}
			}
		case "import":
			for j := 1; j < len(clauseNode.Children); j++ {
				importSpecNode := clauseNode.Children[j]
				if importSpecNode.Type == reader.NodeSymbol {
					fullClassName := importSpecNode.Value
					lastDot := strings.LastIndex(fullClassName, ".")
					if lastDot > 0 && lastDot < len(fullClassName)-1 {

						simpleName := fullClassName[lastDot+1:]

						referredSymbols = append(referredSymbols, ReferredSymbol{SymbolName: simpleName, OriginalNamespace: fullClassName, DefinitionNode: importSpecNode})
					}
				} else if (importSpecNode.Type == reader.NodeList || importSpecNode.Type == reader.NodeVector) && len(importSpecNode.Children) > 0 {

					packageNode := importSpecNode.Children[0]
					if packageNode.Type == reader.NodeSymbol {
						packageName := packageNode.Value
						for k := 1; k < len(importSpecNode.Children); k++ {
							classNode := importSpecNode.Children[k]
							if classNode.Type == reader.NodeSymbol {
								simpleName := classNode.Value
								referredSymbols = append(referredSymbols, ReferredSymbol{SymbolName: simpleName, OriginalNamespace: packageName + "." + simpleName, DefinitionNode: classNode})
							}
						}
					}
				}
			}

		}
	}
	return namespaceName, aliases, referredSymbols, nil
}

// collectTopLevelRequires supports the traditional `(require '[ns :as x])`
// form. It is still common in scripts and in the synthetic catalog even
// though modern namespaces normally place the same libspec in ns.
func collectTopLevelRequires(roots []*reader.RichNode) ([]NamespaceAlias, []ReferredSymbol) {
	var aliases []NamespaceAlias
	var referred []ReferredSymbol
	for _, root := range roots {
		if root == nil || root.Type != reader.NodeList || len(root.Children) < 2 ||
			root.Children[0].Type != reader.NodeSymbol || root.Children[0].Value != "require" {
			continue
		}
		for _, argument := range root.Children[1:] {
			if argument.Type == reader.NodeQuote && len(argument.Children) == 1 {
				argument = argument.Children[0]
			}
			if argument.Type != reader.NodeVector || len(argument.Children) == 0 ||
				argument.Children[0].Type != reader.NodeSymbol {
				continue
			}
			fullNamespace := argument.Children[0].Value
			for i := 1; i+1 < len(argument.Children); i++ {
				option, value := argument.Children[i], argument.Children[i+1]
				if option.Type != reader.NodeKeyword {
					continue
				}
				switch strings.TrimPrefix(option.Value, ":") {
				case "as":
					if value.Type == reader.NodeSymbol {
						aliases = append(aliases, NamespaceAlias{
							Alias: value.Value, FullNamespace: fullNamespace, DefinitionNode: argument,
						})
					}
				case "refer":
					if value.Type == reader.NodeVector {
						for _, symbol := range value.Children {
							if symbol.Type == reader.NodeSymbol {
								referred = append(referred, ReferredSymbol{
									SymbolName: symbol.Value, OriginalNamespace: fullNamespace, DefinitionNode: argument,
								})
							}
						}
					}
				}
				i++
			}
		}
	}
	return aliases, referred
}

var (
	analyzersMu sync.RWMutex
	analyzers   = make(map[*config.Config]*Analyzer)
)

func getOrCreateAnalyzer(cfg *config.Config) *Analyzer {
	analyzersMu.RLock()
	if a, exists := analyzers[cfg]; exists {
		analyzersMu.RUnlock()
		return a
	}
	analyzersMu.RUnlock()

	analyzersMu.Lock()
	defer analyzersMu.Unlock()
	if a, exists := analyzers[cfg]; exists {
		return a
	}
	a := NewAnalyzer(cfg)
	analyzers[cfg] = a
	return a
}

func (a *Analyzer) AnalyzeFile(filepath string) (AnalysisResult, error) {
	tree, err := reader.ParseFile(filepath)
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("parsing file failed: %w", err)
	}

	richRoots, comments := reader.BuildRichTree(tree)

	if EnableExperimentalMacroExpansion {
		ExpandMacros(richRoots)
	}

	var namespaceName string
	var aliases []NamespaceAlias
	var referredSymbols []ReferredSymbol
	var nsNode *reader.RichNode

	for _, root := range richRoots {
		if root.Type == reader.NodeList && len(root.Children) > 0 && root.Children[0].Type == reader.NodeSymbol && root.Children[0].Value == "ns" {
			nsNode = root
			break
		}
	}

	if nsNode != nil {
		var nsParseErr error
		namespaceName, aliases, referredSymbols, nsParseErr = parseNamespaceForm(nsNode)
		if nsParseErr != nil {

		}
	}
	legacyAliases, legacyRefers := collectTopLevelRequires(richRoots)
	aliases = append(aliases, legacyAliases...)
	referredSymbols = append(referredSymbols, legacyRefers...)

	globalScope := NewScope(nil)

	for _, alias := range aliases {
		globalScope.DefineAlias(alias)

		aliasSymInfo := &SymbolInfo{Name: alias.Alias, Definition: alias.DefinitionNode, Type: TypeNamespace}
		globalScope.Define(aliasSymInfo)
	}
	for _, ref := range referredSymbols {
		globalScope.DefineReferredSymbol(ref)

		refSymInfo := &SymbolInfo{
			Name:            ref.SymbolName,
			Definition:      ref.DefinitionNode,
			Type:            TypeReferred,
			OriginNamespace: ref.OriginalNamespace,
		}
		if strings.Contains(ref.OriginalNamespace, ".") && !strings.HasPrefix(ref.OriginalNamespace, "clojure.") {
			refSymInfo.Type = TypeJava
		}
		globalScope.Define(refSymInfo)
	}

	CollectDefinitions(richRoots, globalScope)

	ResolveSymbols(richRoots, globalScope)

	findingsFromAnalysis := a.Analyze(filepath, richRoots, comments, globalScope)

	concreteFindings := make([]rules.Finding, 0, len(findingsFromAnalysis))
	for _, fptr := range findingsFromAnalysis {
		if fptr != nil {
			suppressed := false
			if fptr.Location != nil {
				for _, comment := range comments {
					if comment.Location != nil && (comment.Location.StartLine == fptr.Location.StartLine || comment.Location.StartLine == fptr.Location.StartLine-1) {
						cVal := strings.ToLower(comment.Value)
						if strings.Contains(cVal, "arit:disable-next-line " + fptr.RuleID) || 
						   strings.Contains(cVal, "arit:disable-next-line all") ||
						   strings.Contains(cVal, "clj-kondo/ignore") ||
						   strings.Contains(cVal, "eslint-disable") ||
						   strings.Contains(cVal, "nosonar") {
							suppressed = true
							break
						}
					}
				}
			}
			if !suppressed {
				concreteFindings = append(concreteFindings, *fptr)
			}
		}
	}

	return AnalysisResult{
		Findings:        concreteFindings,
		RichRoots:       richRoots,
		GlobalScope:     globalScope,
		Namespace:       namespaceName,
		Aliases:         aliases,
		ReferredSymbols: referredSymbols,
	}, nil
}

func AnalyzeFile(filepath string, cfg *config.Config) (AnalysisResult, error) {
	if cfg == nil {
		return AnalysisResult{}, fmt.Errorf("configuration cannot be nil")
	}
	analyzerInstance := getOrCreateAnalyzer(cfg)
	return analyzerInstance.AnalyzeFile(filepath)
}
