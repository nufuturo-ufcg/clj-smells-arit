package analyzer

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
	"github.com/thlaurentino/arit/internal/rules/functional"
	"github.com/thlaurentino/arit/internal/rules/semantics"
)

var EnableExperimentalMacroExpansion bool
var EnableExperimentalCrossNamespace bool
var EnableExperimentalTypeInference bool
var EnableExperimentalAsyncCFG bool
var EnableTiming bool

// PhaseTimings contains diagnostic-only timings for one file analysis. It is
// populated only when EnableTiming is true and does not affect findings.
type PhaseTimings struct {
	Parse             time.Duration
	BuildRichTree     time.Duration
	MacroExpansion    time.Duration
	Resolution        time.Duration
	FunctionSummaries time.Duration
	SemanticFacts     time.Duration
	RuleTraversal     time.Duration
	Postprocess       time.Duration
	ProjectIndex      time.Duration
	FileDiscovery     time.Duration
}

func (t *PhaseTimings) Add(other PhaseTimings) {
	if t == nil {
		return
	}
	t.Parse += other.Parse
	t.BuildRichTree += other.BuildRichTree
	t.MacroExpansion += other.MacroExpansion
	t.Resolution += other.Resolution
	t.FunctionSummaries += other.FunctionSummaries
	t.SemanticFacts += other.SemanticFacts
	t.RuleTraversal += other.RuleTraversal
	t.Postprocess += other.Postprocess
	t.ProjectIndex += other.ProjectIndex
	t.FileDiscovery += other.FileDiscovery
}

type AnalysisResult struct {
	Findings            []rules.Finding
	SemanticDiagnostics rules.SemanticDiagnosticsReport
	RichRoots           []*reader.RichNode
	GlobalScope         *Scope
	Namespace           string
	Aliases             []NamespaceAlias
	ReferredSymbols     []ReferredSymbol
	Timings             PhaseTimings
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
	TypeRecord          SymbolType = "record"
)

type SymbolInfo struct {
	Name            string
	Definition      *reader.RichNode
	BindingValue    *reader.RichNode
	Type            SymbolType
	IsPrivate       bool
	IsUsed          bool
	OriginNamespace string
	TypeHint        string
	InferredType    string
}

// The semantic package intentionally does not import analyzer. These methods
// expose the package-independent portion of a local binding through a small
// structural interface consumed by semantic facts.
func (s *SymbolInfo) SemanticTypeHint() string {
	if s == nil {
		return ""
	}
	return s.TypeHint
}

func (s *SymbolInfo) SemanticInferredType() string {
	if s == nil {
		return ""
	}
	return s.InferredType
}

func (s *SymbolInfo) SemanticBindingValue() *reader.RichNode {
	if s == nil {
		return nil
	}
	return s.BindingValue
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
	IsJavaClass       bool
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
	if !s.cacheValid {
		s.mu.Unlock()
		return
	}

	s.cacheValid = false

	if s.lookupCache != nil {
		s.lookupCache = nil
	}
	parent := s.parent
	s.mu.Unlock()

	if parent != nil {
		parent.invalidateCache()
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

			case "let", "loop", "if-let", "when-let":
				if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeVector {
					bindingsNode := node.Children[1]
					letScope := NewScope(currentScope)
					nextScope = letScope

					for i := 0; i < len(bindingsNode.Children); i += 2 {
						if i+1 >= len(bindingsNode.Children) {
							break
						}
						bindingVarNode := bindingsNode.Children[i]
						bindingValNode := bindingsNode.Children[i+1]
						// let/loop bindings are sequential: an initializer sees
						// only bindings defined to its left.
						visit(bindingValNode, letScope)
						if node.Children[0].Value == "loop" {
							defineBindingForm(bindingVarNode, letScope, localDefs, TypeVariable)
						} else {
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
			case "defrecord":
				if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeSymbol {
					recordSymbolNode := node.Children[1]
					recordInfo := &SymbolInfo{
						Name:         recordSymbolNode.Value,
						Definition:   node,
						Type:         TypeRecord,
						IsUsed:       false,
						InferredType: "Record",
					}
					currentScope.Define(recordInfo)
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
			if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil && isBindingScopeForm(node.Children[0].Value) {
				if idx == 1 && child.Type == reader.NodeVector {
					isLetLoopBindingVector = true
				} else if idx > 1 {
					currentChildScope = nextScope
					if node.Children[0].Value == "if-let" && idx >= 3 {
						currentChildScope = currentScope
					}
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

		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0] != nil &&
			node.Children[0].Type == reader.NodeSymbol &&
			(node.Children[0].Value == "defmulti" || node.Children[0].Value == "defmethod") {
			node.Children[0].Resolution = resolveSymbolIdentity(node.Children[0], currentScope)
		}

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

			case "let", "loop", "if-let", "when-let":
				if len(node.Children) > 1 && node.Children[1].Type == reader.NodeVector {
					newLetScope := NewScope(currentScope)
					bindingsNode := node.Children[1]
					for i := 0; i < len(bindingsNode.Children); i += 2 {
						if i+1 < len(bindingsNode.Children) {
							bindingVarNode := bindingsNode.Children[i]
							bindingValNode := bindingsNode.Children[i+1]
							// Resolve the initializer before introducing its own
							// binding, matching Clojure's sequential let semantics.
							visit(bindingValNode, newLetScope)
							if node.Children[0].Value == "loop" {
								defineBindingForm(bindingVarNode, newLetScope, nil, TypeVariable)
							} else {
								defineBindingFormWithValue(bindingVarNode, bindingValNode, newLetScope, nil, TypeVariable)
							}
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

			if node.Type == reader.NodeList && len(node.Children) > 0 && isBindingScopeForm(node.Children[0].Value) {
				if idx == 1 && child.Type == reader.NodeVector {
					continue
				} else if idx > 1 {
					currentChildScope = nextScope
					if node.Children[0].Value == "if-let" && idx >= 3 {
						currentChildScope = currentScope
					}
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
	if info, found := scope.Lookup(name); found {
		if info != nil && info.Type == TypeJava && info.OriginNamespace != "" {
			return info.OriginNamespace, true
		}
		return "", false
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
		resolution.Lexical = info.Definition != nil && info.Definition.Type == reader.NodeSymbol
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
	case reader.NodeFnLiteral:
		return rules.ExecutionDeferred
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

func (a *Analyzer) Analyze(filepath string, richRootNodes []*reader.RichNode, comments []*reader.RichNode, globalScope *Scope, namespaceName string, projectIndex *semantics.ProjectIndex, timings *PhaseTimings) ([]*rules.Finding, rules.SemanticDiagnosticsReport) {

	var findingsMutex sync.Mutex
	allFindings := []*rules.Finding{}
	diagnostics := rules.NewSemanticDiagnostics()
	shadowedCoreCache := make(map[*Scope]map[string]bool)

	var traverseAndAnalyze func(node *reader.RichNode, currentContext map[string]interface{}, scope *Scope)
	traverseAndAnalyze = func(node *reader.RichNode, currentContext map[string]interface{}, scope *Scope) {
		if node == nil {
			return
		}

		prevScope := currentContext["scope"]
		prevConfig := currentContext["config"]
		prevShadowedCore := currentContext["shadowed-core"]
		currentContext["scope"] = scope
		currentContext["config"] = a.Config
		currentContext["shadowed-core"] = cachedShadowedCoreSymbols(shadowedCoreCache, scope)
		semanticFacts := semantics.ForNode(node, currentContext)
		previousSemanticFacts := currentContext["semantic-facts"]
		currentContext["semantic-facts"] = &semanticFacts

		for _, rule := range a.Rules {
			if finding := rule.Check(node, currentContext, filepath); finding != nil {
				finding.Generated = node.Generated
				if node.Origin != nil {
					finding.OriginLocation = node.Origin
				}
				rules.MarkContextualFinding(finding)
				diagnostics.RecordFinding(finding)
				finding.SourceRole = rules.FileRole(currentContext)
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
			node.Type == reader.NodeReaderDiscard || node.Type == reader.NodeFnLiteral {
			if enclosing, ok := currentContext["enclosingForms"].([]string); ok {
				prevEnclosing = enclosing
				marker := "__non-evaluated__"
				if node.Type == reader.NodeFnLiteral {
					marker = "__fn-literal__"
				}
				currentContext["enclosingForms"] = append(enclosing, marker)
				enclosingChanged = true
			}
		}

		parentIsInsideFunc, _ := currentContext["isInsideFunction"].(bool)
		parentIsInsideLet, _ := currentContext["isInsideLet"].(bool)
		parentIsInsideLoop, _ := currentContext["isInsideLoop"].(bool)
		parentIsInsideBinding, _ := currentContext["isInsideBinding"].(bool)
		parentIsInsideDosync, _ := currentContext["isInsideDosync"].(bool)
		parentIsInsideWithOpen, _ := currentContext["isInsideWithOpen"].(bool)
		parentIsInCaseConstant, _ := currentContext["isInCaseConstantPosition"].(bool)

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
			case "let", "if-let", "when-let":
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

			if node.Type == reader.NodeList && len(node.Children) > 0 && isBindingScopeForm(node.Children[0].Value) {
				if idx == 1 && child.Type == reader.NodeVector {
					currentChildScope = scope
				} else if node.Children[0].Value == "if-let" && idx >= 3 {
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
			if currentNodeDefinesLet && node.Children[0].Value == "if-let" && idx >= 3 {
				childIsInsideLet = parentIsInsideLet
			}
			childIsInsideLoop := parentIsInsideLoop || (currentNodeDefinesLoop && idx > 0)
			childIsInsideBinding := parentIsInsideBinding || (currentNodeDefinesBinding && idx > 0)
			childIsInsideDosync := parentIsInsideDosync || (currentNodeDefinesDosync && idx > 0)
			childIsInsideWithOpen := parentIsInsideWithOpen || (currentNodeDefinesWithOpen && idx > 0)
			childIsInCaseConstant := parentIsInCaseConstant
			if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol &&
				node.Children[0].Value == "case" && idx >= 2 && idx%2 == 0 {
				childIsInCaseConstant = true
			}

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

			prevChildCaseConstant := currentContext["isInCaseConstantPosition"]
			currentContext["isInCaseConstantPosition"] = childIsInCaseConstant

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
			currentContext["isInCaseConstantPosition"] = prevChildCaseConstant
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
		currentContext["shadowed-core"] = prevShadowedCore
		currentContext["semantic-facts"] = previousSemanticFacts
	}

	initialContext := map[string]interface{}{
		"isInEagerContext":         false,
		"isInsideFunction":         false,
		"isInsideLet":              false,
		"isInsideLoop":             false,
		"isInsideBinding":          false,
		"isInsideDosync":           false,
		"isInsideWithOpen":         false,
		"isInCaseConstantPosition": false,
		"executionContext":         rules.ExecutionAtLoad,
		"semantic-options": map[string]bool{
			"cross-namespace": EnableExperimentalCrossNamespace,
			"type-inference":  EnableExperimentalTypeInference,
			"async-cfg":       EnableExperimentalAsyncCFG,
			"macro-expansion": EnableExperimentalMacroExpansion,
		},
		"current-namespace":    namespaceName,
		"file-role":            classifyFileRole(filepath),
		"semantic-contracts":   a.Config.SemanticContracts,
		"enclosingForms":       make([]string, 0, 32),
		"ancestorNodes":        make([]*reader.RichNode, 0, 32),
		"semantic-facts-cache": semantics.NewFactsCache(),
		"semantic-diagnostics": diagnostics,
		"dynamic-vars":         make(map[string]bool),
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
		"namespace-requires": func() map[string]bool {
			required := make(map[string]bool)
			if globalScope != nil {
				for _, alias := range globalScope.aliases {
					if alias != nil && alias.FullNamespace != "" {
						required[alias.FullNamespace] = true
					}
				}
				for _, referred := range globalScope.referredSymbols {
					if referred != nil && referred.OriginalNamespace != "" {
						required[referred.OriginalNamespace] = true
					}
				}
			}
			return required
		}(),
	}
	var summaryStart time.Time
	if EnableTiming && timings != nil {
		summaryStart = time.Now()
	}
	initialContext["function-summaries"] = semantics.BuildFunctionSummaries(richRootNodes, namespaceName)
	if EnableTiming && timings != nil {
		timings.FunctionSummaries += time.Since(summaryStart)
		initialContext["semantic-facts-timing"] = &timings.SemanticFacts
	}
	if projectIndex != nil {
		initialContext["project-index"] = projectIndex
	}

	var ruleStart time.Time
	if EnableTiming && timings != nil {
		ruleStart = time.Now()
	}
	topLevelAfterExecutable := false
	for _, rootNode := range richRootNodes {
		initialContext["top-level-after-executable"] = topLevelAfterExecutable
		traverseAndAnalyze(rootNode, initialContext, globalScope)
		if isTopLevelExecutableForm(rootNode) {
			topLevelAfterExecutable = true
		}
	}

	initialContext["scope"] = globalScope
	initialContext["config"] = a.Config

	for _, commentNode := range comments {
		for _, rule := range a.Rules {
			if finding := rule.Check(commentNode, initialContext, filepath); finding != nil {
				finding.Generated = commentNode.Generated
				if commentNode.Origin != nil {
					finding.OriginLocation = commentNode.Origin
				}
				rules.MarkContextualFinding(finding)
				diagnostics.RecordFinding(finding)
				finding.SourceRole = rules.FileRole(initialContext)
				findingsMutex.Lock()
				allFindings = append(allFindings, finding)
				findingsMutex.Unlock()
			}
		}
	}
	if EnableTiming && timings != nil {
		timings.RuleTraversal += time.Since(ruleStart)
	}

	delete(initialContext, "scope")
	delete(initialContext, "config")

	return allFindings, diagnostics.Report()
}

func shadowedCoreSymbols(scope *Scope) map[string]bool {
	shadowed := make(map[string]bool)
	for name := range coreSymbols {
		info, found := scope.Lookup(name)
		if !found || info == nil {
			continue
		}
		if info.Type != TypeCoreFunction && info.Type != TypeCoreSpecialForm {
			shadowed[name] = true
		}
	}
	return shadowed
}

func cachedShadowedCoreSymbols(cache map[*Scope]map[string]bool, scope *Scope) map[string]bool {
	if cached, found := cache[scope]; found {
		return cached
	}
	shadowed := shadowedCoreSymbols(scope)
	cache[scope] = shadowed
	return shadowed
}

func isTopLevelExecutableForm(node *reader.RichNode) bool {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 || node.Children[0] == nil || node.Children[0].Type != reader.NodeSymbol {
		return false
	}
	name := strings.TrimPrefix(node.Children[0].Value, "clojure.core/")
	switch name {
	case "ns", "require", "use", "import", "load-file":
		return false
	default:
		return true
	}
}

// classifyFileRole gives rules a conservative signal about source provenance.
// It is intentionally path-based: the analyzer does not need to understand a
// build tool in order to avoid treating generated fixtures as production code.
func classifyFileRole(path string) string {
	normalized := filepath.ToSlash(path)
	if strings.Contains(normalized, "/internal/test/data/") {
		return "test-fixture"
	}
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	for _, part := range parts {
		switch strings.ToLower(part) {
		case "target":
			return "generated"
		case "generated", "fixtures", "fixture", "testcases", "test-resources", "corpus", "examples", "example", "benchmark", "benchmarks", "expanded_smells_catalog", "synthetic-catalog":
			return "fixture"
		case "tests", "test", "int-test", "integration-test":
			return "test"
		case "dev", "development", "notebook", "notebooks", "clerk", "clay":
			return "dev"
		case "dev-resources", "development-resources", "repl":
			return "dev"
		case "build", "scripts", "script", "bin":
			return "build"
		}
	}
	base := strings.ToLower(filepath.Base(normalized))
	if strings.HasSuffix(base, "_test.clj") || strings.HasSuffix(base, "-test.clj") {
		return "test"
	}
	if base == "project.clj" || base == "deps.edn" || base == "build.clj" {
		return "build"
	}
	return "production"
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
	defineBindingPatternWithValue(bindingNode, valNode, targetScope, localDefs, defaultSymbolType)
}

func defineBindingPatternWithValue(bindingNode *reader.RichNode, valNode *reader.RichNode, targetScope *Scope, localDefs map[*reader.RichNode]*SymbolInfo, defaultSymbolType SymbolType) {
	if bindingNode == nil || targetScope == nil {
		return
	}
	if bindingNode.Type == reader.NodeTag {
		if len(bindingNode.Children) == 1 && bindingNode.Children[0] != nil {
			bindingNode.Children[0].TypeHint = bindingNode.Value
			defineBindingPatternWithValue(bindingNode.Children[0], valNode, targetScope, localDefs, defaultSymbolType)
		}
		return
	}
	if bindingNode.Type == reader.NodeSymbol {
		symbolName := bindingNode.Value
		if symbolName == "_" || symbolName == "&" || strings.HasPrefix(symbolName, ".") || strings.Contains(symbolName, "/") {
			return
		}
		info := &SymbolInfo{
			Name:         symbolName,
			Definition:   bindingNode,
			BindingValue: valNode,
			Type:         defaultSymbolType,
			IsUsed:       false,
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

	switch bindingNode.Type {
	case reader.NodeVector:
		sourceNode := bindingSourceValue(valNode)
		if sourceNode == nil || sourceNode.Type != reader.NodeVector {
			defineBindingForm(bindingNode, targetScope, localDefs, defaultSymbolType)
			return
		}
		sourceIndex := 0
		for i := 0; i < len(bindingNode.Children); i++ {
			pattern := bindingNode.Children[i]
			if pattern == nil {
				continue
			}
			if pattern.Type == reader.NodeKeyword && pattern.Value == ":as" {
				if i+1 < len(bindingNode.Children) {
					defineBindingPatternWithValue(bindingNode.Children[i+1], valNode, targetScope, localDefs, defaultSymbolType)
				}
				i++
				continue
			}
			if pattern.Type == reader.NodeSymbol && pattern.Value == "&" {
				// A rest binding needs sequence semantics, not just an AST slice.
				// Keep it lexical-only until that contract is modeled explicitly.
				if i+1 < len(bindingNode.Children) {
					defineBindingForm(bindingNode.Children[i+1], targetScope, localDefs, defaultSymbolType)
				}
				i++
				continue
			}
			var projected *reader.RichNode
			if sourceIndex < len(sourceNode.Children) {
				projected = sourceNode.Children[sourceIndex]
			}
			defineBindingPatternWithValue(pattern, projected, targetScope, localDefs, defaultSymbolType)
			sourceIndex++
		}
	case reader.NodeMap:
		sourceNode := bindingSourceValue(valNode)
		if sourceNode == nil || sourceNode.Type != reader.NodeMap || len(sourceNode.Children)%2 != 0 {
			defineBindingForm(bindingNode, targetScope, localDefs, defaultSymbolType)
			return
		}
		for i := 0; i+1 < len(bindingNode.Children); i += 2 {
			keyNode := bindingNode.Children[i]
			patternNode := bindingNode.Children[i+1]
			if keyNode == nil || patternNode == nil || keyNode.Type != reader.NodeKeyword {
				if sourceValue, found := literalMapValue(sourceNode, keyNode); found {
					defineBindingPatternWithValue(patternNode, sourceValue, targetScope, localDefs, defaultSymbolType)
				} else if keyNode != nil && keyNode.Type != reader.NodeKeyword {
					defineBindingForm(patternNode, targetScope, localDefs, defaultSymbolType)
				}
				continue
			}
			switch strings.TrimPrefix(keyNode.Value, ":") {
			case "as":
				defineBindingPatternWithValue(patternNode, sourceNode, targetScope, localDefs, defaultSymbolType)
			case "keys", "strs", "syms":
				if patternNode.Type != reader.NodeVector {
					continue
				}
				for _, nameNode := range patternNode.Children {
					if nameNode == nil || nameNode.Type != reader.NodeSymbol {
						continue
					}
					lookupKey := &reader.RichNode{Type: reader.NodeKeyword, Value: ":" + nameNode.Value}
					switch strings.TrimPrefix(keyNode.Value, ":") {
					case "strs":
						lookupKey = &reader.RichNode{Type: reader.NodeString, Value: nameNode.Value}
					case "syms":
						lookupKey = &reader.RichNode{Type: reader.NodeSymbol, Value: nameNode.Value}
					}
					if sourceValue, found := literalMapValue(sourceNode, lookupKey); found {
						defineBindingPatternWithValue(nameNode, sourceValue, targetScope, localDefs, defaultSymbolType)
					} else {
						defineBindingForm(nameNode, targetScope, localDefs, defaultSymbolType)
					}
				}
			case "or":
				// Defaults require presence/absence semantics and are deliberately
				// not promoted by this structural summary.
			default:
				if sourceValue, found := literalMapValue(sourceNode, keyNode); found {
					defineBindingPatternWithValue(patternNode, sourceValue, targetScope, localDefs, defaultSymbolType)
				} else {
					defineBindingForm(patternNode, targetScope, localDefs, defaultSymbolType)
				}
			}
		}
	default:
		defineBindingForm(bindingNode, targetScope, localDefs, defaultSymbolType)
	}
}

func bindingSourceValue(node *reader.RichNode) *reader.RichNode {
	if node == nil || node.Type != reader.NodeSymbol || node.SymbolRef == nil {
		return node
	}
	if info, ok := node.SymbolRef.(*SymbolInfo); ok && info != nil && info.BindingValue != nil {
		return bindingSourceValue(info.BindingValue)
	}
	return node
}

func literalMapValue(node, key *reader.RichNode) (*reader.RichNode, bool) {
	if node == nil || node.Type != reader.NodeMap || key == nil || len(node.Children)%2 != 0 {
		return nil, false
	}
	var found *reader.RichNode
	for i := 0; i+1 < len(node.Children); i += 2 {
		candidate := node.Children[i]
		if candidate != nil && candidate.Type == key.Type && candidate.Value == key.Value {
			found = node.Children[i+1]
		}
	}
	return found, found != nil
}

func defineBindingForm(bindingNode *reader.RichNode, targetScope *Scope, localDefs map[*reader.RichNode]*SymbolInfo, defaultSymbolType SymbolType) {
	if bindingNode == nil {
		return
	}

	switch bindingNode.Type {
	case reader.NodeTag:
		if len(bindingNode.Children) == 1 && bindingNode.Children[0] != nil {
			bindingNode.Children[0].TypeHint = bindingNode.Value
			defineBindingForm(bindingNode.Children[0], targetScope, localDefs, defaultSymbolType)
		}

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
				switch strings.TrimPrefix(keyNode.Value, ":") {
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

func isBindingScopeForm(name string) bool {
	switch name {
	case "let", "loop", "if-let", "when-let":
		return true
	default:
		return false
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

		case "let", "loop", "if-let", "when-let":
			if childIndex == 1 && childNode.Type == reader.NodeVector {
				return true
			}
		case "def", "defonce", "defrecord":

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

type namespaceItems struct {
	aliases  []NamespaceAlias
	referred []ReferredSymbol
}

func parseNamespaceClause(clauseNode *reader.RichNode) namespaceItems {
	items := namespaceItems{}
	if clauseNode == nil || clauseNode.Type != reader.NodeList || len(clauseNode.Children) == 0 || clauseNode.Children[0] == nil || clauseNode.Children[0].Type != reader.NodeKeyword {
		return items
	}

	switch strings.TrimPrefix(clauseNode.Children[0].Value, ":") {
	case "require":
		for j := 1; j < len(clauseNode.Children); j++ {
			specNode := clauseNode.Children[j]
			if specNode == nil || specNode.Type != reader.NodeVector || len(specNode.Children) == 0 || specNode.Children[0] == nil || specNode.Children[0].Type != reader.NodeSymbol {
				continue
			}
			fullNs := specNode.Children[0].Value
			var currentAlias string
			var refers []string
			for k := 1; k < len(specNode.Children); k++ {
				optionKeyNode := specNode.Children[k]
				if optionKeyNode == nil || optionKeyNode.Type != reader.NodeKeyword || k+1 >= len(specNode.Children) {
					continue
				}
				optionValueNode := specNode.Children[k+1]
				k++
				switch strings.TrimPrefix(optionKeyNode.Value, ":") {
				case "as":
					if optionValueNode != nil && optionValueNode.Type == reader.NodeSymbol {
						currentAlias = optionValueNode.Value
					}
				case "refer":
					if optionValueNode != nil && optionValueNode.Type == reader.NodeVector {
						for _, referSymNode := range optionValueNode.Children {
							if referSymNode != nil && referSymNode.Type == reader.NodeSymbol {
								refers = append(refers, referSymNode.Value)
							}
						}
					}
				}
			}
			if currentAlias != "" {
				items.aliases = append(items.aliases, NamespaceAlias{Alias: currentAlias, FullNamespace: fullNs, DefinitionNode: specNode})
			}
			for _, referSym := range refers {
				items.referred = append(items.referred, ReferredSymbol{SymbolName: referSym, OriginalNamespace: fullNs, DefinitionNode: specNode})
			}
		}
	case "import":
		for j := 1; j < len(clauseNode.Children); j++ {
			importSpecNode := clauseNode.Children[j]
			if importSpecNode == nil {
				continue
			}
			if importSpecNode.Type == reader.NodeSymbol {
				fullClassName := importSpecNode.Value
				lastDot := strings.LastIndex(fullClassName, ".")
				if lastDot > 0 && lastDot < len(fullClassName)-1 {
					items.referred = append(items.referred, ReferredSymbol{SymbolName: fullClassName[lastDot+1:], OriginalNamespace: fullClassName, DefinitionNode: importSpecNode, IsJavaClass: true})
				}
			} else if (importSpecNode.Type == reader.NodeList || importSpecNode.Type == reader.NodeVector) && len(importSpecNode.Children) > 0 && importSpecNode.Children[0] != nil && importSpecNode.Children[0].Type == reader.NodeSymbol {
				packageName := importSpecNode.Children[0].Value
				for k := 1; k < len(importSpecNode.Children); k++ {
					classNode := importSpecNode.Children[k]
					if classNode != nil && classNode.Type == reader.NodeSymbol {
						items.referred = append(items.referred, ReferredSymbol{SymbolName: classNode.Value, OriginalNamespace: packageName + "." + classNode.Value, DefinitionNode: classNode, IsJavaClass: true})
					}
				}
			}
		}
	}
	return items
}

func mergeNamespaceItems(dst *namespaceItems, src namespaceItems) {
	dst.aliases = append(dst.aliases, src.aliases...)
	dst.referred = append(dst.referred, src.referred...)
}

func commonNamespaceItems(branches []namespaceItems) namespaceItems {
	if len(branches) == 0 {
		return namespaceItems{}
	}
	var common namespaceItems
	for _, candidate := range branches[0].aliases {
		matchesEveryBranch := true
		for _, branch := range branches[1:] {
			matches := 0
			for _, alias := range branch.aliases {
				if alias.Alias == candidate.Alias && alias.FullNamespace == candidate.FullNamespace {
					matches++
				}
			}
			if matches != 1 {
				matchesEveryBranch = false
				break
			}
		}
		if matchesEveryBranch {
			common.aliases = append(common.aliases, candidate)
		}
	}
	for _, candidate := range branches[0].referred {
		matchesEveryBranch := true
		for _, branch := range branches[1:] {
			matches := 0
			for _, ref := range branch.referred {
				if ref.SymbolName == candidate.SymbolName && ref.OriginalNamespace == candidate.OriginalNamespace && ref.IsJavaClass == candidate.IsJavaClass {
					matches++
				}
			}
			if matches != 1 {
				matchesEveryBranch = false
				break
			}
		}
		if matchesEveryBranch {
			common.referred = append(common.referred, candidate)
		}
	}
	return common
}

func collectNamespaceConditional(node *reader.RichNode) namespaceItems {
	if node == nil || (node.Type != reader.NodeReaderCond && node.Type != reader.NodeReaderCondSplice) {
		return namespaceItems{}
	}
	branches := make([]namespaceItems, 0, len(node.Children)/2)
	for i := 1; i < len(node.Children); i += 2 {
		branch := namespaceItems{}
		mergeNamespaceItems(&branch, parseNamespaceClause(node.Children[i]))
		branches = append(branches, branch)
	}
	return commonNamespaceItems(branches)
}

func parseNamespaceForm(nsNode *reader.RichNode) (string, []NamespaceAlias, []ReferredSymbol, error) {
	if nsNode == nil || nsNode.Type != reader.NodeList || len(nsNode.Children) == 0 || nsNode.Children[0].Value != "ns" {
		return "", nil, nil, fmt.Errorf("node is not a valid ns form")
	}

	var namespaceName string
	var aliases []NamespaceAlias
	var referredSymbols []ReferredSymbol
	nameIndex := -1
	for i := 1; i < len(nsNode.Children); i++ {
		if nsNode.Children[i] != nil && nsNode.Children[i].Type == reader.NodeSymbol {
			nameIndex = i
			namespaceName = nsNode.Children[i].Value
			break
		}
	}

	clauseStart := nameIndex + 1
	if clauseStart < 1 {
		clauseStart = 1
	}
	for i := clauseStart; i < len(nsNode.Children); i++ {
		clauseNode := nsNode.Children[i]
		if clauseNode == nil {
			continue
		}
		if clauseNode.Type == reader.NodeReaderCond || clauseNode.Type == reader.NodeReaderCondSplice {
			items := collectNamespaceConditional(clauseNode)
			aliases = append(aliases, items.aliases...)
			referredSymbols = append(referredSymbols, items.referred...)
			continue
		}
		items := parseNamespaceClause(clauseNode)
		aliases = append(aliases, items.aliases...)
		referredSymbols = append(referredSymbols, items.referred...)
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
	return a.AnalyzeFileWithProjectIndex(filepath, nil)
}

func (a *Analyzer) AnalyzeFileWithProjectIndex(filepath string, projectIndex *semantics.ProjectIndex) (AnalysisResult, error) {
	timings := PhaseTimings{}
	var phaseStart time.Time
	var richRoots []*reader.RichNode
	var comments []*reader.RichNode
	reusedIndexedFile := false
	if projectIndex != nil {
		richRoots, comments, reusedIndexedFile = projectIndex.CachedFile(filepath)
	}
	if !reusedIndexedFile {
		if EnableTiming {
			phaseStart = time.Now()
		}
		tree, err := reader.ParseFile(filepath)
		if err != nil {
			return AnalysisResult{}, fmt.Errorf("parsing file failed: %w", err)
		}
		if EnableTiming {
			timings.Parse = time.Since(phaseStart)
			phaseStart = time.Now()
		}

		richRoots, comments = reader.BuildRichTree(tree)
		if EnableTiming {
			timings.BuildRichTree = time.Since(phaseStart)
		}
	}

	if EnableExperimentalMacroExpansion {
		// The project index owns the parsed AST. Experimental expansion must use
		// a derived tree so it cannot mutate shared cross-namespace state.
		richRoots = cloneRichRoots(richRoots)
		if EnableTiming {
			phaseStart = time.Now()
		}
		ExpandMacros(richRoots)
		if EnableTiming {
			timings.MacroExpansion = time.Since(phaseStart)
		}
	}
	if EnableTiming {
		phaseStart = time.Now()
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
		if ref.IsJavaClass || (strings.Contains(ref.OriginalNamespace, ".") && !strings.HasPrefix(ref.OriginalNamespace, "clojure.")) {
			refSymInfo.Type = TypeJava
		}
		globalScope.Define(refSymInfo)
	}

	CollectDefinitions(richRoots, globalScope)

	ResolveSymbols(richRoots, globalScope)
	if projectIndex != nil {
		projectIndex.EnrichResolutions(richRoots)
	}
	if EnableTiming {
		timings.Resolution = time.Since(phaseStart)
	}

	findingsFromAnalysis, semanticDiagnostics := a.Analyze(filepath, richRoots, comments, globalScope, namespaceName, projectIndex, &timings)

	if EnableTiming {
		phaseStart = time.Now()
	}
	concreteFindings := make([]rules.Finding, 0, len(findingsFromAnalysis))
	for _, fptr := range findingsFromAnalysis {
		if fptr != nil {
			suppressed := false
			if fptr.Location != nil {
				for _, comment := range comments {
					if comment.Location != nil && (comment.Location.StartLine == fptr.Location.StartLine || comment.Location.StartLine == fptr.Location.StartLine-1) {
						cVal := strings.ToLower(comment.Value)
						if strings.Contains(cVal, "arit:disable-next-line "+fptr.RuleID) ||
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
	if EnableTiming {
		timings.Postprocess = time.Since(phaseStart)
	}

	return AnalysisResult{
		Findings:            concreteFindings,
		SemanticDiagnostics: semanticDiagnostics,
		RichRoots:           richRoots,
		GlobalScope:         globalScope,
		Namespace:           namespaceName,
		Aliases:             aliases,
		ReferredSymbols:     referredSymbols,
		Timings:             timings,
	}, nil
}

func cloneRichRoots(roots []*reader.RichNode) []*reader.RichNode {
	seen := make(map[*reader.RichNode]*reader.RichNode)
	result := make([]*reader.RichNode, 0, len(roots))
	for _, root := range roots {
		result = append(result, cloneRichNode(root, seen))
	}
	return result
}

func cloneRichNode(node *reader.RichNode, seen map[*reader.RichNode]*reader.RichNode) *reader.RichNode {
	if node == nil {
		return nil
	}
	if cloned, ok := seen[node]; ok {
		return cloned
	}
	cloned := *node
	cloned.Children = nil
	cloned.Metadata = nil
	cloned.Comments = nil
	cloned.ResolvedDefinition = nil
	cloned.Scope = nil
	cloned.SymbolRef = nil
	cloned.Resolution = nil
	seen[node] = &cloned
	for _, child := range node.Children {
		cloned.Children = append(cloned.Children, cloneRichNode(child, seen))
	}
	cloned.Metadata = cloneRichNode(node.Metadata, seen)
	for _, comment := range node.Comments {
		cloned.Comments = append(cloned.Comments, cloneRichNode(comment, seen))
	}
	return &cloned
}

func AnalyzeFile(filepath string, cfg *config.Config) (AnalysisResult, error) {
	if cfg == nil {
		return AnalysisResult{}, fmt.Errorf("configuration cannot be nil")
	}
	analyzerInstance := getOrCreateAnalyzer(cfg)
	return analyzerInstance.AnalyzeFile(filepath)
}
