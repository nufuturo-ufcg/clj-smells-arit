package analyzer

import (
	"fmt"

	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/config"
	"github.com/thlaurentino/arit/internal/reader"
	"github.com/thlaurentino/arit/internal/rules"
)

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
			case "defn", "defn-":
				if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeSymbol {
					funcSymbolNode := node.Children[1]
					funcInfo := &SymbolInfo{
						Name:       funcSymbolNode.Value,
						Definition: node,
						Type:       TypeFunction,
						IsPrivate:  funcNameNode.Value == "defn-",
						IsUsed:     false,
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
						if paramsNodeCandidate.Type == reader.NodeVector {
							defineParams(paramsNodeCandidate, fnScope, localDefs)
						} else if paramsNodeCandidate.Type == reader.NodeList {
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
					if paramsNodeCandidate.Type == reader.NodeVector {
						defineParams(paramsNodeCandidate, fnScope, localDefs)
					} else if paramsNodeCandidate.Type == reader.NodeList {
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
							defineBindingForm(bindingVarNode, letScope, localDefs, TypeVariable)
						}
					}
				}
			case "def", "defonce":
				if len(node.Children) > 1 && node.Children[1] != nil && node.Children[1].Type == reader.NodeSymbol {
					varSymbolNode := node.Children[1]
					varInfo := &SymbolInfo{
						Name:       varSymbolNode.Value,
						Definition: node,
						Type:       TypeVariable,
						IsUsed:     false,
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
			case "defn", "defn-":
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
						if paramsNode.Type == reader.NodeVector {
							defineParams(paramsNode, newFnScope, nil)
						} else if paramsNode.Type == reader.NodeList {
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
					if paramsNode.Type == reader.NodeVector {
						defineParams(paramsNode, newFnScope, nil)
					} else if paramsNode.Type == reader.NodeList {
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
							defineBindingForm(bindingVarNode, newLetScope, nil, TypeVariable)
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

			} else if aliasInfo, aliasFound := currentScope.findAlias(symbolName); aliasFound {
				node.SymbolRef = aliasInfo

			} else {

			}
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

type Analyzer struct {
	Rules []rules.CheckerRule
}

func NewAnalyzer(cfg *config.Config) *Analyzer {
	analyzer := &Analyzer{}

	allRuleInstances := rules.AllRules()

	for _, ruleInst := range allRuleInstances {

		if _ = ruleInst.(rules.CheckerRule); true {

		}
	}

	for _, ruleInstance := range allRuleInstances {
		checkerRule, ok := ruleInstance.(rules.CheckerRule)
		if !ok {

			continue
		}

		ruleMetaID := checkerRule.Meta().ID

		isEnabledInGlobalConfig, specifiedInGlobalConfig := cfg.EnabledRules[ruleMetaID]
		shouldProcessRule := !specifiedInGlobalConfig || isEnabledInGlobalConfig

		if !shouldProcessRule {

			continue
		}

		configuredRule := configureRule(checkerRule, cfg)

		analyzer.Rules = append(analyzer.Rules, configuredRule)

	}

	return analyzer
}

func configureRule(rule rules.CheckerRule, cfg *config.Config) rules.CheckerRule {
	ruleMetaID := rule.Meta().ID
	ruleCfg, cfgExists := cfg.RuleConfig[ruleMetaID]

	if !cfgExists {
		return rule
	}

	if typedRule, ok := rule.(*rules.LazySideEffectsRule); ok {
		newRule := &rules.LazySideEffectsRule{
			LazyContextFuncs: make(map[string]bool),
			SideEffectFuncs:  make(map[string]bool),
		}

		for k, v := range rules.DefaultLazyContextFunctions {
			newRule.LazyContextFuncs[k] = v
		}
		for k, v := range rules.DefaultSideEffectFunctions {
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
	_, isEager := rules.EagerConsumerFunctions[funcNode.Value]
	return isEager
}

func (a *Analyzer) Analyze(filepath string, richRootNodes []*reader.RichNode, comments []*reader.RichNode, globalScope *Scope) []*rules.Finding {

	var findingsMutex sync.Mutex
	allFindings := []*rules.Finding{}

	var traverseAndAnalyze func(node *reader.RichNode, currentContext map[string]interface{}, scope *Scope)
	traverseAndAnalyze = func(node *reader.RichNode, currentContext map[string]interface{}, scope *Scope) {
		if node == nil {
			return
		}

		for _, rule := range a.Rules {
			ruleContext := make(map[string]interface{})
			for k, v := range currentContext {
				ruleContext[k] = v
			}
			ruleContext["scope"] = scope

			if finding := rule.Check(node, ruleContext, filepath); finding != nil {
				findingsMutex.Lock()
				allFindings = append(allFindings, finding)
				findingsMutex.Unlock()
			}
		}

		nextScope := scope

		childContext := make(map[string]interface{})
		for k, v := range currentContext {
			childContext[k] = v
		}
		childContext["parent"] = node

		isParentEager, _ := currentContext["isInEagerContext"].(bool)
		childContext["isInEagerContext"] = isParentEager || isNodeEagerConsumer(node)

		parentIsInsideFunc, _ := currentContext["isInsideFunction"].(bool)
		parentIsInsideLet, _ := currentContext["isInsideLet"].(bool)
		parentIsInsideLoop, _ := currentContext["isInsideLoop"].(bool)
		parentIsInsideBinding, _ := currentContext["isInsideBinding"].(bool)
		parentIsInsideDosync, _ := currentContext["isInsideDosync"].(bool)

		currentNodeDefinesFunc := false
		currentNodeDefinesLet := false
		currentNodeDefinesLoop := false
		currentNodeDefinesBinding := false
		currentNodeDefinesDosync := false

		if node.Type == reader.NodeList && len(node.Children) > 0 && node.Children[0].Type == reader.NodeSymbol {
			nodeVal := node.Children[0].Value
			switch nodeVal {
			case "defn", "defn-", "fn":
				currentNodeDefinesFunc = true
			case "let":
				currentNodeDefinesLet = true
			case "loop":
				currentNodeDefinesLoop = true
			case "binding":
				currentNodeDefinesBinding = true
			case "dosync":
				currentNodeDefinesDosync = true
			}
		}

		for idx, child := range node.Children {
			currentChildScope := nextScope

			if node.Type == reader.NodeList && len(node.Children) > 0 && (node.Children[0].Value == "let" || node.Children[0].Value == "loop") {
				if idx == 1 && child.Type == reader.NodeVector {
					currentChildScope = scope
				}
			}

			traversalContext := make(map[string]interface{})
			for k, v := range childContext {
				traversalContext[k] = v
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
			traversalContext["isInsideFunction"] = childIsInsideFunc

			childIsInsideLet := parentIsInsideLet
			if currentNodeDefinesLet && idx > 0 {
				childIsInsideLet = true
			}
			traversalContext["isInsideLet"] = childIsInsideLet

			childIsInsideLoop := parentIsInsideLoop
			if currentNodeDefinesLoop && idx > 0 {
				childIsInsideLoop = true
			}
			traversalContext["isInsideLoop"] = childIsInsideLoop

			childIsInsideBinding := parentIsInsideBinding
			if currentNodeDefinesBinding && idx > 0 {
				childIsInsideBinding = true
			}
			traversalContext["isInsideBinding"] = childIsInsideBinding

			childIsInsideDosync := parentIsInsideDosync
			if currentNodeDefinesDosync && idx > 0 {
				childIsInsideDosync = true
			}
			traversalContext["isInsideDosync"] = childIsInsideDosync

			traverseAndAnalyze(child, traversalContext, currentChildScope)
		}

	}

	initialContext := map[string]interface{}{
		"isInEagerContext": false,
		"isInsideFunction": false,
		"isInsideLet":      false,
		"isInsideLoop":     false,
		"isInsideBinding":  false,
		"isInsideDosync":   false,
	}

	for _, rootNode := range richRootNodes {
		traverseAndAnalyze(rootNode, initialContext, globalScope)
	}

	for _, commentNode := range comments {
		for _, rule := range a.Rules {
			ruleContext := make(map[string]interface{})
			for k, v := range initialContext {
				ruleContext[k] = v
			}
			ruleContext["scope"] = globalScope

			if finding := rule.Check(commentNode, ruleContext, filepath); finding != nil {
				findingsMutex.Lock()
				allFindings = append(allFindings, finding)
				findingsMutex.Unlock()
			}
		}
	}
	
	for _, rootNode := range richRootNodes {
		PrintRichNode(rootNode, "")
	}

	return allFindings
}

func PrintRichNode(node *reader.RichNode, indent string) {
	if node == nil {
		return
	}

	if node.Value != "" {
		fmt.Printf("%s%s: %s\n", indent, node.Type, node.Value)
	} else {
		fmt.Printf("%s%s\n", indent, node.Type)
	}

	for _, child := range node.Children {
		PrintRichNode(child, indent+"  ")
	}


}

func defineParams(paramsNode *reader.RichNode, targetScope *Scope, localDefs map[*reader.RichNode]*SymbolInfo) {
	if paramsNode == nil || paramsNode.Type != reader.NodeVector {
		return
	}
	defineBindingForm(paramsNode, targetScope, localDefs, TypeParameter)
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
			Name:       symbolName,
			Definition: bindingNode,
			Type:       defaultSymbolType,
			IsUsed:     false,
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
		case "defn", "defn-":

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
		clauseKeyword := clauseNode.Children[0].Value

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
					optionKey := optionKeyNode.Value
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

func AnalyzeFile(filepath string, cfg *config.Config) (AnalysisResult, error) {
	tree, err := reader.ParseFile(filepath)
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("parsing file failed: %w", err)
	}

	richRoots, comments := reader.BuildRichTree(tree)

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

	if cfg == nil {
		return AnalysisResult{}, fmt.Errorf("configuration cannot be nil")
	}
	analyzerInstance := NewAnalyzer(cfg)
	findingsFromAnalysis := analyzerInstance.Analyze(filepath, richRoots, comments, globalScope)

	concreteFindings := make([]rules.Finding, 0, len(findingsFromAnalysis))
	for _, fptr := range findingsFromAnalysis {
		if fptr != nil {
			concreteFindings = append(concreteFindings, *fptr)
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
