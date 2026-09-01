// Package semantics contains package-independent semantic facts shared by
// analysis rules. It intentionally depends only on the reader package so
// rules can consume these facts without creating import cycles.
package semantics

import (
	"fmt"
	"strings"
	"time"

	"github.com/thlaurentino/arit/internal/reader"
)

type Evidence string

const (
	EvidenceProven    Evidence = "proven"
	EvidenceUnknown   Evidence = "unknown"
	EvidenceDisproved Evidence = "disproved"
)

// EvidenceSource records where a fact came from. A source is deliberately
// descriptive; it does not upgrade an unknown or contextual fact by itself.
type EvidenceSource string

const (
	SourceUnknown             EvidenceSource = "unknown"
	SourceSyntaxLiteral       EvidenceSource = "syntax-literal"
	SourceLexicalResolution   EvidenceSource = "lexical-resolution"
	SourceCanonicalResolution EvidenceSource = "canonical-resolution"
	SourceProjectIndex        EvidenceSource = "project-index"
	SourceFunctionSummary     EvidenceSource = "function-summary"
	SourceConfiguredContract  EvidenceSource = "configured-contract"
	SourceLocalBinding        EvidenceSource = "local-binding"
)

type AbstractType string

const (
	TypeUnknown   AbstractType = "unknown"
	TypeNil       AbstractType = "nil"
	TypeBoolean   AbstractType = "boolean"
	TypeNumber    AbstractType = "number"
	TypeString    AbstractType = "string"
	TypeKeyword   AbstractType = "keyword"
	TypeSymbol    AbstractType = "symbol"
	TypeCharacter AbstractType = "character"
	TypeRegex     AbstractType = "regex"
	TypeList      AbstractType = "list"
	TypeVector    AbstractType = "vector"
	TypeMap       AbstractType = "map"
	TypeSet       AbstractType = "set"
	TypeFunction  AbstractType = "function"
	TypeAtom      AbstractType = "atom"
	TypeRef       AbstractType = "ref"
	TypeChannel   AbstractType = "channel"
	TypeJava      AbstractType = "java-object"
)

type Nilability string

const (
	NilabilityUnknown Nilability = "unknown"
	Nilable           Nilability = "nilable"
	NonNil            Nilability = "non-nil"
)

type Laziness string

const (
	LazinessUnknown Laziness = "unknown"
	Lazy            Laziness = "lazy"
	Eager           Laziness = "eager"
)

type CallMode string

const (
	CallModeUnknown    CallMode = "unknown"
	CallModeCollection CallMode = "collection"
	CallModeTransducer CallMode = "transducer"
)

type DataPosition string

const (
	DataPositionUnknown DataPosition = "unknown"
	DataPositionFirst   DataPosition = "first"
	DataPositionLast    DataPosition = "last"
	DataPositionSingle  DataPosition = "single"
)

type ExecutionPhase string

const (
	ExecutionUnknown      ExecutionPhase = "unknown"
	ExecutionAtLoad       ExecutionPhase = "load"
	ExecutionDeferred     ExecutionPhase = "deferred"
	ExecutionNonEvaluated ExecutionPhase = "non-evaluated"
)

type Effect uint64

const (
	EffectNone    Effect = 0
	EffectUnknown Effect = 1 << iota
	EffectIO
	EffectBlocking
	EffectMutation
	EffectDynamicBinding
	EffectChannelSend
	EffectChannelClose
	EffectAsyncBoundary
	EffectNamespaceLoad
)

func (e Effect) Has(value Effect) bool {
	return e&value != 0
}

type Facts struct {
	Node              *reader.RichNode
	Resolution        *reader.SymbolResolution
	Type              AbstractType
	Nilability        Nilability
	Laziness          Laziness
	Effects           Effect
	Execution         ExecutionPhase
	CallMode          CallMode
	DataPosition      DataPosition
	DataArgumentIndex int
	Literal           bool
	Constant          bool
	ConstantValue     string
	ArgumentCount     int
	ArityKnown        bool
	ArityValid        bool
	ResolutionKnown   bool
	ResolutionSource  EvidenceSource
	SemanticEvidence  Evidence
	EvidenceSource    EvidenceSource
	Generated         bool
	Origin            *reader.Location
}

// FactsCache stores facts for one AST analysis. It is intentionally scoped by
// the analyzer to a single file: resolutions, summaries, contracts and the
// project index are all analysis inputs and must never leak across files or
// configurations.
//
// The analyzer traverses one file serially, so the cache does not use a
// mutex. Callers that share a cache across concurrent traversals must provide
// separate caches instead.
type FactsCache struct {
	entries        map[factsCacheKey]Facts
	literalEntries map[*reader.RichNode]literalValueResult
}

type literalValueResult struct {
	literal bool
	value   string
}

type factsCacheKey struct {
	node          *reader.RichNode
	execution     ExecutionPhase
	typeInference bool
	projectIndex  *ProjectIndex
	hasSummaries  bool
	hasContracts  bool
}

func NewFactsCache() *FactsCache {
	return &FactsCache{
		entries:        make(map[factsCacheKey]Facts),
		literalEntries: make(map[*reader.RichNode]literalValueResult),
	}
}

func factsCacheFromContext(context map[string]interface{}) *FactsCache {
	if context == nil {
		return nil
	}
	cache, _ := context["semantic-facts-cache"].(*FactsCache)
	return cache
}

func (c *FactsCache) key(node *reader.RichNode, context map[string]interface{}) factsCacheKey {
	var projectIndex *ProjectIndex
	if context != nil {
		projectIndex, _ = context["project-index"].(*ProjectIndex)
	}
	_, hasSummaries := contextValue[map[string]FunctionSummary](context, "function-summaries")
	_, hasContracts := contextValue[map[string]map[string]interface{}](context, "semantic-contracts")
	return factsCacheKey{
		node:          node,
		execution:     executionPhase(context),
		typeInference: semanticOption(context, "type-inference"),
		projectIndex:  projectIndex,
		hasSummaries:  hasSummaries,
		hasContracts:  hasContracts,
	}
}

func contextValue[T any](context map[string]interface{}, name string) (T, bool) {
	var zero T
	if context == nil {
		return zero, false
	}
	value, ok := context[name].(T)
	return value, ok
}

// FunctionContract is an explicit, opt-in contract for a function that is
// outside the canonical operation table. A contract only proves fields that
// are declared and validated; its presence never proves omitted fields.
type FunctionContract struct {
	Effects                Effect
	EffectsKnown           bool
	ReturnsType            AbstractType
	ReturnsTypeKnown       bool
	Laziness               Laziness
	LazinessKnown          bool
	CallMode               CallMode
	CallModeKnown          bool
	DataPosition           DataPosition
	DataArgumentIndex      int
	DataPositionKnown      bool
	DataArgumentIndexKnown bool
	AllowedArities         []int
	ArityKnown             bool
	Valid                  bool
}

func ForNode(node *reader.RichNode, context map[string]interface{}) Facts {
	var timing *time.Duration
	if context != nil {
		timing, _ = context["semantic-facts-timing"].(*time.Duration)
	}
	var started time.Time
	if timing != nil {
		started = time.Now()
		defer func() { *timing += time.Since(started) }()
	}
	if cache := factsCacheFromContext(context); cache != nil {
		if cache.entries == nil {
			cache.entries = make(map[factsCacheKey]Facts)
		}
		if cache.literalEntries == nil {
			cache.literalEntries = make(map[*reader.RichNode]literalValueResult)
		}
		key := cache.key(node, context)
		if facts, found := cache.entries[key]; found {
			return facts
		}
		facts := computeFacts(node, context)
		cache.entries[key] = facts
		return facts
	}
	return computeFacts(node, context)
}

func computeFacts(node *reader.RichNode, context map[string]interface{}) Facts {
	facts := Facts{
		Node:              node,
		Type:              TypeUnknown,
		Nilability:        NilabilityUnknown,
		Laziness:          LazinessUnknown,
		Execution:         executionPhase(context),
		CallMode:          CallModeUnknown,
		DataPosition:      DataPositionUnknown,
		DataArgumentIndex: -1,
		SemanticEvidence:  EvidenceUnknown,
		EvidenceSource:    SourceUnknown,
		ResolutionSource:  SourceUnknown,
	}
	if node == nil {
		return facts
	}

	facts.Resolution = node.Resolution
	facts.Generated = node.Generated
	facts.Origin = node.Origin
	facts.ResolutionKnown = node.Resolution != nil && node.Resolution.Kind != reader.ResolutionUnresolved
	if (node.Type == reader.NodeList || node.Type == reader.NodeFnLiteral) && len(node.Children) > 0 && node.Children[0] != nil {
		facts.Resolution = node.Children[0].Resolution
		facts.ResolutionKnown = facts.Resolution != nil && facts.Resolution.Kind != reader.ResolutionUnresolved
	}
	facts.ResolutionSource = resolutionSource(facts.Resolution, context)
	facts.Type = nodeType(node, semanticOption(context, "type-inference"))
	facts.Literal, facts.ConstantValue = constantValueCached(node, factsCacheFromContext(context))
	facts.Constant = facts.Literal
	if facts.Constant {
		facts.Nilability = NonNil
		if node.Type == reader.NodeNil {
			facts.Nilability = Nilable
		}
	}
	if node.Type == reader.NodeSymbol && node.Resolution != nil && node.Resolution.Kind == reader.ResolutionLocal {
		facts.SemanticEvidence = EvidenceProven
		facts.EvidenceSource = SourceLexicalResolution
		facts = enrichLocalBindingFacts(facts, node, context)
	}

	if node.Type == reader.NodeList || node.Type == reader.NodeFnLiteral {
		facts = enrichCallFacts(facts, node, context)
		facts = enrichArityFacts(facts, node)
		facts = enrichConfiguredContractFacts(facts, node, context)
	}
	if facts.Constant && facts.SemanticEvidence == EvidenceUnknown {
		facts.SemanticEvidence = EvidenceProven
		facts.EvidenceSource = SourceSyntaxLiteral
	}
	return facts
}

// semanticBinding is the package-independent contract exposed by the
// analyzer for local bindings. Keeping this structural avoids an import cycle
// while allowing semantic facts to reuse information already computed during
// lexical resolution.
type semanticBinding interface {
	SemanticTypeHint() string
	SemanticInferredType() string
	SemanticBindingValue() *reader.RichNode
}

func enrichLocalBindingFacts(facts Facts, node *reader.RichNode, context map[string]interface{}) Facts {
	if node == nil || node.SymbolRef == nil {
		return facts
	}
	binding, ok := node.SymbolRef.(semanticBinding)
	if !ok {
		return facts
	}

	if semanticOption(context, "type-inference") {
		if inferred := normalizeType(binding.SemanticTypeHint()); inferred != TypeUnknown {
			facts.Type = inferred
		} else if inferred := normalizeType(binding.SemanticInferredType()); inferred != TypeUnknown {
			facts.Type = inferred
		}
	}

	value := binding.SemanticBindingValue()
	if value == nil {
		return facts
	}
	valueFacts := ForNode(value, context)
	if valueFacts.SemanticEvidence != EvidenceProven {
		return facts
	}

	if valueFacts.Constant {
		facts.Literal = true
		facts.Constant = true
		facts.ConstantValue = valueFacts.ConstantValue
		facts.Nilability = valueFacts.Nilability
	}
	if valueFacts.Type != TypeUnknown {
		facts.Type = valueFacts.Type
	}
	if valueFacts.Laziness != LazinessUnknown {
		facts.Laziness = valueFacts.Laziness
	}
	if valueFacts.Constant || valueFacts.Type != TypeUnknown || valueFacts.Laziness != LazinessUnknown {
		facts.SemanticEvidence = EvidenceProven
		facts.EvidenceSource = SourceLocalBinding
	}
	return facts
}

func IsCall(node *reader.RichNode, canonicalName string) bool {
	if node == nil || len(node.Children) == 0 || node.Children[0] == nil || node.Children[0].Resolution == nil {
		return false
	}
	return node.Children[0].Resolution.CanonicalName == canonicalName
}

func CallName(node *reader.RichNode) string {
	if node == nil || len(node.Children) == 0 || node.Children[0] == nil {
		return ""
	}
	if node.Children[0].Resolution != nil && node.Children[0].Resolution.CanonicalName != "" {
		return node.Children[0].Resolution.CanonicalName
	}
	return node.Children[0].Value
}

func nodeType(node *reader.RichNode, useInference bool) AbstractType {
	if node == nil {
		return TypeUnknown
	}
	if useInference {
		if node.TypeHint != "" {
			if inferred := normalizeType(node.TypeHint); inferred != TypeUnknown {
				return inferred
			}
		}
		if node.InferredType != "" {
			if inferred := normalizeType(node.InferredType); inferred != TypeUnknown {
				return inferred
			}
		}
	}
	switch node.Type {
	case reader.NodeNil:
		return TypeNil
	case reader.NodeBool:
		return TypeBoolean
	case reader.NodeNumber:
		return TypeNumber
	case reader.NodeString:
		return TypeString
	case reader.NodeKeyword:
		return TypeKeyword
	case reader.NodeSymbol:
		return TypeSymbol
	case reader.NodeCharacter:
		return TypeCharacter
	case reader.NodeRegex:
		return TypeRegex
	case reader.NodeList, reader.NodeFnLiteral:
		return TypeList
	case reader.NodeVector:
		return TypeVector
	case reader.NodeMap:
		return TypeMap
	case reader.NodeSet:
		return TypeSet
	default:
		return TypeUnknown
	}
}

func semanticOption(context map[string]interface{}, name string) bool {
	if context == nil {
		return false
	}
	options, _ := context["semantic-options"].(map[string]bool)
	return options[name]
}

func resolutionSource(resolution *reader.SymbolResolution, context map[string]interface{}) EvidenceSource {
	if resolution == nil || resolution.Kind == reader.ResolutionUnresolved {
		return SourceUnknown
	}
	if resolution.Kind == reader.ResolutionLocal {
		return SourceLexicalResolution
	}
	if index, ok := context["project-index"].(*ProjectIndex); ok && index != nil && resolution.Namespace != "" && index.HasNamespace(resolution.Namespace) {
		return SourceProjectIndex
	}
	return SourceCanonicalResolution
}

func normalizeType(value string) AbstractType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "nil", "null":
		return TypeNil
	case "bool", "boolean":
		return TypeBoolean
	case "number", "integer", "long", "double", "float":
		return TypeNumber
	case "string":
		return TypeString
	case "keyword":
		return TypeKeyword
	case "symbol":
		return TypeSymbol
	case "char", "character":
		return TypeCharacter
	case "regex":
		return TypeRegex
	case "list":
		return TypeList
	case "vector":
		return TypeVector
	case "map":
		return TypeMap
	case "set":
		return TypeSet
	case "function", "fn":
		return TypeFunction
	case "atom":
		return TypeAtom
	case "ref":
		return TypeRef
	case "channel":
		return TypeChannel
	case "java", "java-object":
		return TypeJava
	default:
		return TypeUnknown
	}
}

func constantValue(node *reader.RichNode) (bool, string) {
	return constantValueCached(node, nil)
}

// constantValueCached evaluates collection literalness bottom-up. Primitive
// nodes use the fast path and never touch the cache; collection nodes cache
// their result after all children have been evaluated. This avoids repeated
// recursive walks when ForNode is requested for both a collection and its
// descendants while keeping the common non-collection path allocation-free.
func constantValueCached(node *reader.RichNode, cache *FactsCache) (bool, string) {
	if node == nil {
		return false, ""
	}
	switch node.Type {
	case reader.NodeKeyword, reader.NodeString, reader.NodeNumber, reader.NodeBool,
		reader.NodeNil, reader.NodeCharacter, reader.NodeRegex:
		return true, node.Value
	case reader.NodeVector, reader.NodeMap, reader.NodeSet:
		if cache != nil {
			if result, found := cache.literalEntries[node]; found {
				return result.literal, result.value
			}
		}
		result := literalValueResult{literal: true, value: node.Value}
		for _, child := range node.Children {
			if child == nil {
				continue
			}
			if constant, _ := constantValueCached(child, cache); !constant {
				result.literal = false
				result.value = ""
				break
			}
		}
		if cache != nil {
			cache.literalEntries[node] = result
		}
		return result.literal, result.value
	default:
		return false, ""
	}
}

func executionPhase(context map[string]interface{}) ExecutionPhase {
	if context == nil {
		return ExecutionUnknown
	}
	switch fmt.Sprint(context["executionContext"]) {
	case "load":
		return ExecutionAtLoad
	case "deferred":
		return ExecutionDeferred
	case "non-evaluated":
		return ExecutionNonEvaluated
	default:
		return ExecutionUnknown
	}
}

func enrichCallFacts(facts Facts, node *reader.RichNode, context map[string]interface{}) Facts {
	name := CallName(node)
	summaryApplied := false
	if summaries, ok := context["function-summaries"].(map[string]FunctionSummary); ok {
		candidates := []string{name}
		if node != nil && len(node.Children) > 0 && node.Children[0] != nil {
			candidates = append(candidates, node.Children[0].Value)
		}
		for _, candidate := range candidates {
			if summary, found := summaries[candidate]; found {
				facts.Effects = summary.EagerEffects
				if summary.ReturnsType != TypeUnknown {
					facts.Type = summary.ReturnsType
				}
				if summary.ReturnsLazy {
					facts.Laziness = Lazy
				}
				if summary.EagerEvidence == EvidenceUnknown {
					facts.Effects |= EffectUnknown
				}
				facts.EvidenceSource = SourceFunctionSummary
				summaryApplied = true
				break
			}
		}
	}
	switch name {
	case "clojure.core/map", "clojure.core/filter", "clojure.core/remove", "clojure.core/keep", "clojure.core/for":
		facts.Laziness = Lazy
		facts.Type = TypeList
	case "clojure.core/mapv", "clojure.core/filterv", "clojure.core/keepv", "clojure.core/into", "clojure.core/vec", "clojure.core/doall":
		facts.Laziness = Eager
		facts.Type = TypeVector
	case "clojure.core/future", "clojure.core/future-call", "clojure.core/send", "clojure.core/send-off":
		facts.Effects |= EffectAsyncBoundary
	case "clojure.core/def", "clojure.core/defonce", "clojure.core/set!", "clojure.core/reset!", "clojure.core/swap!", "clojure.core/alter-var-root":
		facts.Effects |= EffectMutation
	case "clojure.core/aset", "clojure.core/aset-boolean", "clojure.core/aset-byte", "clojure.core/aset-char", "clojure.core/aset-double", "clojure.core/aset-float", "clojure.core/aset-int", "clojure.core/aset-long", "clojure.core/aset-short":
		facts.Effects |= EffectMutation
	case "clojure.core/slurp", "clojure.java.io/input-stream", "clojure.java.io/output-stream":
		facts.Effects |= EffectIO
	case "clojure.core/require", "clojure.core/use", "clojure.core/load", "clojure.core/in-ns", "clojure.core/import":
		facts.Effects |= EffectNamespaceLoad
	case "clojure.core.async/>!", "clojure.core.async/>!!", "clojure.core.async/put!", "clojure.core.async/offer!":
		facts.Effects |= EffectChannelSend
	case "clojure.core.async/close!":
		facts.Effects |= EffectChannelClose
	case "clojure.core.async/<!", "clojure.core.async/<!!", "clojure.core.async/take!":
		facts.Effects |= EffectBlocking
	}
	if !summaryApplied && (facts.Resolution == nil || !facts.ResolutionKnown) {
		if node.Type == reader.NodeList && len(node.Children) > 0 {
			facts.Effects |= EffectUnknown
		}
	}
	if facts.Effects.Has(EffectUnknown) {
		facts.SemanticEvidence = EvidenceUnknown
		facts.EvidenceSource = SourceUnknown
	} else if facts.Effects != EffectNone || facts.Laziness != LazinessUnknown {
		facts.SemanticEvidence = EvidenceProven
		if facts.EvidenceSource == SourceUnknown {
			facts.EvidenceSource = SourceCanonicalResolution
		}
	}
	return facts
}

func enrichConfiguredContractFacts(facts Facts, node *reader.RichNode, context map[string]interface{}) Facts {
	if facts.Resolution != nil && (facts.Resolution.Kind == reader.ResolutionClojureCore || facts.Resolution.Kind == reader.ResolutionLocal) {
		return facts
	}
	contract, found := configuredFunctionContract(CallName(node), context)
	if !found || !contract.Valid {
		return facts
	}

	if contract.ArityKnown {
		facts.ArgumentCount = len(node.Children) - 1
		facts.ArityKnown = true
		facts.ArityValid = false
		for _, arity := range contract.AllowedArities {
			if facts.ArgumentCount == arity {
				facts.ArityValid = true
				break
			}
		}
		if !facts.ArityValid {
			facts.Effects |= EffectUnknown
			facts.SemanticEvidence = EvidenceUnknown
			facts.EvidenceSource = SourceUnknown
			return facts
		}
	}
	if contract.EffectsKnown {
		facts.Effects = contract.Effects
	}
	if contract.ReturnsTypeKnown {
		facts.Type = contract.ReturnsType
	}
	if contract.LazinessKnown {
		facts.Laziness = contract.Laziness
	}
	if contract.CallModeKnown {
		facts.CallMode = contract.CallMode
	}
	if contract.DataPositionKnown {
		facts.DataPosition = contract.DataPosition
		if len(node.Children) <= 1 {
			facts.Effects |= EffectUnknown
			facts.SemanticEvidence = EvidenceUnknown
			facts.EvidenceSource = SourceUnknown
			return facts
		}
		switch contract.DataPosition {
		case DataPositionFirst, DataPositionSingle:
			facts.DataArgumentIndex = 0
		case DataPositionLast:
			facts.DataArgumentIndex = len(node.Children) - 2
		}
	}
	if contract.DataArgumentIndexKnown {
		if contract.DataArgumentIndex >= len(node.Children)-1 {
			facts.Effects |= EffectUnknown
			facts.SemanticEvidence = EvidenceUnknown
			facts.EvidenceSource = SourceUnknown
			return facts
		}
		facts.DataArgumentIndex = contract.DataArgumentIndex
	}

	facts.EvidenceSource = SourceConfiguredContract
	if facts.Effects.Has(EffectUnknown) {
		facts.SemanticEvidence = EvidenceUnknown
		facts.EvidenceSource = SourceUnknown
		return facts
	}
	if contract.EffectsKnown || contract.ReturnsTypeKnown || contract.LazinessKnown ||
		contract.CallModeKnown || contract.DataPositionKnown || contract.ArityKnown {
		facts.SemanticEvidence = EvidenceProven
	}
	return facts
}

func configuredFunctionContract(name string, context map[string]interface{}) (FunctionContract, bool) {
	contracts, ok := context["semantic-contracts"].(map[string]map[string]interface{})
	if !ok || name == "" {
		return FunctionContract{}, false
	}
	raw, ok := contracts[name]
	if !ok {
		return FunctionContract{}, false
	}
	contract := FunctionContract{DataPosition: DataPositionUnknown, DataArgumentIndex: -1}
	recognized := false
	for key, value := range raw {
		switch key {
		case "effects":
			effects, valid := configuredEffects(value)
			if !valid {
				return FunctionContract{}, false
			}
			contract.Effects, contract.EffectsKnown, recognized = effects, true, true
		case "returns-type":
			text, valid := value.(string)
			if !valid || normalizeType(text) == TypeUnknown {
				return FunctionContract{}, false
			}
			contract.ReturnsType, contract.ReturnsTypeKnown, recognized = normalizeType(text), true, true
		case "laziness":
			text, valid := value.(string)
			if !valid {
				return FunctionContract{}, false
			}
			switch strings.ToLower(strings.TrimSpace(text)) {
			case "lazy":
				contract.Laziness = Lazy
			case "eager":
				contract.Laziness = Eager
			default:
				return FunctionContract{}, false
			}
			contract.LazinessKnown, recognized = true, true
		case "call-mode":
			text, valid := value.(string)
			if !valid {
				return FunctionContract{}, false
			}
			switch strings.ToLower(strings.TrimSpace(text)) {
			case "collection":
				contract.CallMode = CallModeCollection
			case "transducer":
				contract.CallMode = CallModeTransducer
			default:
				return FunctionContract{}, false
			}
			contract.CallModeKnown, recognized = true, true
		case "data-position":
			text, valid := value.(string)
			if !valid {
				return FunctionContract{}, false
			}
			switch strings.ToLower(strings.TrimSpace(text)) {
			case "first":
				contract.DataPosition = DataPositionFirst
			case "last":
				contract.DataPosition = DataPositionLast
			case "single":
				contract.DataPosition = DataPositionSingle
			default:
				return FunctionContract{}, false
			}
			contract.DataPositionKnown, recognized = true, true
		case "data-argument-index":
			index, valid := configuredInt(value)
			if !valid || index < 0 {
				return FunctionContract{}, false
			}
			contract.DataArgumentIndex, contract.DataArgumentIndexKnown, recognized = index, true, true
		case "arity":
			arities, valid := configuredArities(value)
			if !valid {
				return FunctionContract{}, false
			}
			contract.AllowedArities, contract.ArityKnown, recognized = arities, true, true
		default:
			return FunctionContract{}, false
		}
	}
	contract.Valid = recognized
	return contract, contract.Valid
}

func configuredEffects(value interface{}) (Effect, bool) {
	var values []interface{}
	switch typed := value.(type) {
	case string:
		values = []interface{}{typed}
	case []string:
		for _, item := range typed {
			values = append(values, item)
		}
	case []interface{}:
		values = typed
	default:
		return EffectNone, false
	}
	var effects Effect
	for _, item := range values {
		name, ok := item.(string)
		if !ok {
			return EffectNone, false
		}
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "none", "pure":
		case "io":
			effects |= EffectIO
		case "blocking":
			effects |= EffectBlocking
		case "mutation":
			effects |= EffectMutation
		case "dynamic-binding":
			effects |= EffectDynamicBinding
		case "channel-send":
			effects |= EffectChannelSend
		case "channel-close":
			effects |= EffectChannelClose
		case "async-boundary":
			effects |= EffectAsyncBoundary
		case "namespace-load":
			effects |= EffectNamespaceLoad
		default:
			return EffectNone, false
		}
	}
	return effects, true
}

func configuredArities(value interface{}) ([]int, bool) {
	var values []interface{}
	switch typed := value.(type) {
	case int:
		values = []interface{}{typed}
	case int64:
		values = []interface{}{typed}
	case float64:
		values = []interface{}{typed}
	case []int:
		for _, item := range typed {
			values = append(values, item)
		}
	case []interface{}:
		values = typed
	default:
		return nil, false
	}
	if len(values) == 0 {
		return nil, false
	}
	arities := make([]int, 0, len(values))
	for _, value := range values {
		arity, valid := configuredInt(value)
		if !valid || arity < 0 {
			return nil, false
		}
		arities = append(arities, arity)
	}
	return arities, true
}

func configuredInt(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), typed == float64(int(typed))
	default:
		return 0, false
	}
}

type aritySpec struct {
	min   int
	max   int // -1 means unbounded
	valid func(int) bool
}

func exactArity(count int) bool { return count >= 0 }

var canonicalArities = map[string]aritySpec{
	"clojure.core/map":            {min: 1, max: -1, valid: exactArity},
	"clojure.core/mapv":           {min: 2, max: -1, valid: exactArity},
	"clojure.core/filter":         {min: 1, max: 2, valid: exactArity},
	"clojure.core/filterv":        {min: 2, max: 2, valid: exactArity},
	"clojure.core/remove":         {min: 1, max: 2, valid: exactArity},
	"clojure.core/keep":           {min: 1, max: 2, valid: exactArity},
	"clojure.core/keep-indexed":   {min: 1, max: 2, valid: exactArity},
	"clojure.core/map-indexed":    {min: 1, max: 2, valid: exactArity},
	"clojure.core/take":           {min: 1, max: 2, valid: exactArity},
	"clojure.core/drop":           {min: 1, max: 2, valid: exactArity},
	"clojure.core/take-while":     {min: 1, max: 2, valid: exactArity},
	"clojure.core/drop-while":     {min: 1, max: 2, valid: exactArity},
	"clojure.core/distinct":       {min: 0, max: 1, valid: exactArity},
	"clojure.core/dedupe":         {min: 0, max: 1, valid: exactArity},
	"clojure.core/into":           {min: 2, max: 3, valid: exactArity},
	"clojure.core/doall":          {min: 1, max: 2, valid: exactArity},
	"clojure.core/vec":            {min: 1, max: 1, valid: exactArity},
	"clojure.core/count":          {min: 1, max: 1, valid: exactArity},
	"clojure.core/set":            {min: 1, max: 1, valid: exactArity},
	"clojure.core/keys":           {min: 1, max: 1, valid: exactArity},
	"clojure.core/vals":           {min: 1, max: 1, valid: exactArity},
	"clojure.core/sort":           {min: 1, max: 2, valid: exactArity},
	"clojure.core/sort-by":        {min: 2, max: 3, valid: exactArity},
	"clojure.core/hash-map":       {min: 0, max: -1, valid: func(count int) bool { return count%2 == 0 }},
	"clojure.core/list":           {min: 0, max: -1, valid: exactArity},
	"clojure.core/vector":         {min: 0, max: -1, valid: exactArity},
	"clojure.core/str":            {min: 0, max: -1, valid: exactArity},
	"clojure.core/get":            {min: 2, max: 3, valid: exactArity},
	"clojure.core/contains?":      {min: 2, max: 2, valid: exactArity},
	"clojure.core/assoc":          {min: 3, max: -1, valid: func(count int) bool { return count%2 == 1 }},
	"clojure.core/dissoc":         {min: 2, max: -1, valid: exactArity},
	"clojure.core/conj":           {min: 2, max: -1, valid: exactArity},
	"clojure.core/reduce":         {min: 2, max: 3, valid: exactArity},
	"clojure.core/swap!":          {min: 2, max: -1, valid: exactArity},
	"clojure.core/reset!":         {min: 2, max: 2, valid: exactArity},
	"clojure.core/atom":           {min: 1, max: -1, valid: func(count int) bool { return count%2 == 1 }},
	"clojure.core/ref":            {min: 1, max: -1, valid: func(count int) bool { return count%2 == 1 }},
	"clojure.core/agent":          {min: 1, max: -1, valid: func(count int) bool { return count%2 == 1 }},
	"clojure.core/volatile!":      {min: 1, max: 1, valid: exactArity},
	"clojure.core/alter-var-root": {min: 2, max: -1, valid: exactArity},
	"clojure.core/future":         {min: 0, max: -1, valid: exactArity},
	"clojure.core/future-call":    {min: 1, max: 1, valid: exactArity},
	"clojure.core/defmulti":       {min: 2, max: -1, valid: exactArity},
	"clojure.core/defmethod":      {min: 4, max: -1, valid: exactArity},
	"clojure.core/slurp":          {min: 1, max: 2, valid: exactArity},
	"clojure.core/spit":           {min: 2, max: -1, valid: exactArity},
	"clojure.core/await":          {min: 1, max: -1, valid: exactArity},
	"Thread/sleep":                {min: 1, max: 1, valid: exactArity},
	"java.lang.Thread/sleep":      {min: 1, max: 1, valid: exactArity},
	"clojure.core.async/put!":     {min: 2, max: -1, valid: exactArity},
	"clojure.core.async/>!":       {min: 2, max: 2, valid: exactArity},
	"clojure.core.async/>!!":      {min: 2, max: 2, valid: exactArity},
	"clojure.core.async/<!":       {min: 1, max: 1, valid: exactArity},
	"clojure.core.async/<!!":      {min: 1, max: 1, valid: exactArity},
	"clojure.core.async/close!":   {min: 1, max: 1, valid: exactArity},
	"clojure.core/=":              {min: 2, max: -1, valid: exactArity},
	"clojure.core/==":             {min: 2, max: -1, valid: exactArity},
	"clojure.core/not=":           {min: 2, max: -1, valid: exactArity},
	"clojure.core/>":              {min: 2, max: -1, valid: exactArity},
	"clojure.core/<":              {min: 2, max: -1, valid: exactArity},
	"clojure.core/>=":             {min: 2, max: -1, valid: exactArity},
	"clojure.core/<=":             {min: 2, max: -1, valid: exactArity},
	"clojure.core/+":              {min: 0, max: -1, valid: exactArity},
	"clojure.core/-":              {min: 1, max: -1, valid: exactArity},
	"clojure.core/*":              {min: 0, max: -1, valid: exactArity},
	"clojure.core//":              {min: 1, max: -1, valid: exactArity},
	"clojure.core/mod":            {min: 2, max: 2, valid: exactArity},
	"clojure.core/rem":            {min: 2, max: 2, valid: exactArity},
	"clojure.core/if":             {min: 2, max: 3, valid: exactArity},
}

// CanonicalArityValid validates the argument count of a canonical operation.
// It is useful to consumers that need to validate a transformed call, such as
// a threading macro after inserting its threaded value.
func CanonicalArityValid(name string, argumentCount int) (known, valid bool) {
	spec, known := canonicalArities[name]
	if !known {
		return false, false
	}
	return true, argumentCount >= spec.min && (spec.max < 0 || argumentCount <= spec.max) && spec.valid(argumentCount)
}

func transducerArity(name string, argumentCount int) bool {
	if argumentCount < 0 {
		return false
	}
	switch name {
	case "clojure.core/map", "clojure.core/filter", "clojure.core/remove",
		"clojure.core/keep", "clojure.core/keep-indexed", "clojure.core/map-indexed",
		"clojure.core/take", "clojure.core/drop", "clojure.core/take-while",
		"clojure.core/drop-while":
		return argumentCount == 1
	case "clojure.core/distinct", "clojure.core/dedupe":
		return argumentCount == 0
	default:
		return false
	}
}

func dataPosition(name string, argumentCount int) (DataPosition, int, bool) {
	if argumentCount <= 0 {
		return DataPositionUnknown, -1, false
	}
	switch name {
	case "clojure.core/map", "clojure.core/mapv", "clojure.core/filter", "clojure.core/filterv",
		"clojure.core/remove", "clojure.core/keep", "clojure.core/keep-indexed", "clojure.core/map-indexed",
		"clojure.core/mapcat", "clojure.core/reduce", "clojure.core/reduce-kv", "clojure.core/sort",
		"clojure.core/sort-by", "clojure.core/group-by", "clojure.core/partition", "clojure.core/take",
		"clojure.core/drop", "clojure.core/take-while", "clojure.core/drop-while", "clojure.core/concat",
		"clojure.core/flatten", "clojure.string/join":
		return DataPositionLast, argumentCount - 1, true
	case "clojure.core/assoc", "clojure.core/dissoc", "clojure.core/update", "clojure.core/merge",
		"clojure.core/select-keys", "clojure.core/get", "clojure.core/get-in", "clojure.core/assoc-in",
		"clojure.core/update-in", "clojure.core/conj", "clojure.string/replace", "clojure.string/split":
		return DataPositionFirst, 0, true
	case "clojure.core/distinct", "clojure.core/vec", "clojure.core/set", "clojure.core/seq",
		"clojure.core/keys", "clojure.core/vals", "clojure.core/first", "clojure.core/last",
		"clojure.core/rest", "clojure.core/next", "clojure.core/reverse", "clojure.core/count",
		"clojure.core/name", "clojure.core/keyword", "clojure.core/identity", "clojure.core/inc",
		"clojure.core/dec", "clojure.string/trim", "clojure.string/upper-case", "clojure.string/lower-case",
		"clojure.string/capitalize":
		if argumentCount == 1 {
			return DataPositionSingle, 0, true
		}
	}
	return DataPositionUnknown, -1, false
}

func enrichArityFacts(facts Facts, node *reader.RichNode) Facts {
	if node == nil || node.Type != reader.NodeList || len(node.Children) == 0 {
		return facts
	}
	name := CallName(node)
	spec, known := canonicalArities[name]
	if !known {
		return facts
	}
	facts.ArgumentCount = len(node.Children) - 1
	facts.ArityKnown = true
	facts.ArityValid = facts.ArgumentCount >= spec.min && (spec.max < 0 || facts.ArgumentCount <= spec.max) && spec.valid(facts.ArgumentCount)
	if !facts.ArityValid {
		facts.Effects |= EffectUnknown
		facts.SemanticEvidence = EvidenceUnknown
		facts.EvidenceSource = SourceUnknown
		return facts
	}
	facts.CallMode = CallModeCollection
	if transducerArity(name, facts.ArgumentCount) {
		facts.CallMode = CallModeTransducer
		facts.Laziness = LazinessUnknown
		facts.Type = TypeFunction
		facts.SemanticEvidence = EvidenceProven
		facts.EvidenceSource = SourceCanonicalResolution
		return facts
	}
	if position, index, known := dataPosition(name, facts.ArgumentCount); known {
		facts.DataPosition = position
		facts.DataArgumentIndex = index
	}
	return facts
}
