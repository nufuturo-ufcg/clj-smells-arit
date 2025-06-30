package rules

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cespare/goclj/parse"
	"github.com/thlaurentino/arit/internal/reader"
)

type ExactDuplicatedCodeAnalyzer struct {
	mu                   sync.Mutex
	codeBlocks           map[string][]ExactCodeBlockInfo
	minLines             int
	minTokens            int
	initialized          bool
	normalizedCache      map[string]string
	cacheMutex           sync.RWMutex
	maxCacheSize         int
	cacheHits            int64
	cacheMisses          int64
	lastCleanup          time.Time
	cleanupInterval      time.Duration
	maxBlocksPerHash     int
	totalBlocksProcessed int64
}

type ExactCodeBlockInfo struct {
	Hash      string
	File      string
	Location  *reader.Location
	Lines     int
	Tokens    int
	BlockType string
	Context   string
}

var exactAnalyzer *ExactDuplicatedCodeAnalyzer
var exactRegexInitOnce sync.Once

var (
	exactNumericLiteralRegex *regexp.Regexp
	exactSymbolPatterns      map[string]*regexp.Regexp
	exactSymbolReplacements  map[string]string
)

func initExactRegexPatterns() {
	exactRegexInitOnce.Do(func() {
		exactNumericLiteralRegex = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
		exactSymbolPatterns = make(map[string]*regexp.Regexp)
		exactSymbolReplacements = make(map[string]string)
		patterns := []struct {
			pattern     string
			replacement string
		}{
			{`^(?:data|info|result|response|request|payload)$`, "DATA_VAR"},
			{`^(?:item|element|entry|record)s?$`, "ITEM_VAR"},
			{`^(?:id|key|index|idx)$`, "ID_VAR"},
			{`^(?:value|val|amount|total|sum)s?$`, "VALUE_VAR"},
			{`^(?:name|title|label)$`, "NAME_VAR"},
			{`^(?:user|customer|person|entity|account)s?$`, "ENTITY_VAR"},
			{`^(?:config|settings|options|params?)$`, "CONFIG_VAR"},
			{`^get-`, "GET_FUNC"},
			{`^set-`, "SET_FUNC"},
			{`^process-`, "PROCESS_FUNC"},
			{`^create-`, "CREATE_FUNC"},
			{`^validate-`, "VALIDATE_FUNC"},
			{`^calculate-`, "CALC_FUNC"},
		}
		for _, p := range patterns {
			compiled := regexp.MustCompile(p.pattern)
			exactSymbolPatterns[p.pattern] = compiled
			exactSymbolReplacements[p.pattern] = p.replacement
		}
	})
}

func GetExactDuplicatedCodeAnalyzer() *ExactDuplicatedCodeAnalyzer {
	if exactAnalyzer == nil {
		initExactRegexPatterns()
		exactAnalyzer = &ExactDuplicatedCodeAnalyzer{
			codeBlocks:       make(map[string][]ExactCodeBlockInfo),
			normalizedCache:  make(map[string]string),
			minLines:         6,
			minTokens:        25,
			maxCacheSize:     10000,
			cleanupInterval:  30 * time.Second,
			maxBlocksPerHash: 10,
		}
	}
	return exactAnalyzer
}

func (e *ExactDuplicatedCodeAnalyzer) AnalyzeTree(tree *parse.Tree, richNodes []*reader.RichNode, filepath string) []Finding {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cleanupIfNeeded()

	var findings []Finding
	blocks := e.extractSignificantBlocksOptimized(richNodes, filepath)

	for _, block := range blocks {

		isDuplicate := false
		if len(e.codeBlocks[block.Hash]) > 0 {
			isDuplicate = true
		}

		e.codeBlocks[block.Hash] = append(e.codeBlocks[block.Hash], block)
		e.totalBlocksProcessed++

		if isDuplicate {

			finding := Finding{
				RuleID:   "duplicated-code-exact",
				Message:  e.createExactDuplicationMessage(block, len(e.codeBlocks[block.Hash])),
				Filepath: filepath,
				Location: block.Location,
				Severity: SeverityWarning,
			}
			findings = append(findings, finding)
		}
	}

	return findings
}

func (e *ExactDuplicatedCodeAnalyzer) extractSignificantBlocksOptimized(nodes []*reader.RichNode, filepath string) []ExactCodeBlockInfo {
	var blocks []ExactCodeBlockInfo
	blockCount := 0
	maxBlocksPerFile := 1000

	for _, node := range nodes {
		if blockCount >= maxBlocksPerFile {
			break
		}
		e.extractBlocksFromNodeOptimized(node, filepath, "", &blocks, &blockCount)
	}
	return blocks
}

func (e *ExactDuplicatedCodeAnalyzer) extractBlocksFromNodeOptimized(node *reader.RichNode, filepath, context string, blocks *[]ExactCodeBlockInfo, blockCount *int) {
	if node == nil || *blockCount >= 1000 {
		return
	}

	blockAdded := false
	switch {
	case e.isFunctionDefinition(node):
		blockAdded = e.extractFunctionBlockOptimized(node, filepath, blocks, blockCount)
	case e.isLetBlock(node):
		blockAdded = e.extractLetBlockOptimized(node, filepath, context, blocks, blockCount)
	case e.isConditionalBlock(node):
		blockAdded = e.extractConditionalBlockOptimized(node, filepath, context, blocks, blockCount)
	case e.isLoopBlock(node):
		blockAdded = e.extractLoopBlockOptimized(node, filepath, context, blocks, blockCount)
	case e.isSignificantBlock(node):
		blockAdded = e.extractGenericBlockOptimized(node, filepath, context, blocks, blockCount)
	}

	if blockAdded {
		return
	}

	newContext := context
	if e.isFunctionDefinition(node) && len(node.Children) > 1 {
		newContext = node.Children[1].Value
	}

	e.processChildrenOptimized(node, filepath, newContext, blocks, blockCount, 0)
}

func (e *ExactDuplicatedCodeAnalyzer) processChildrenOptimized(node *reader.RichNode, filepath, context string, blocks *[]ExactCodeBlockInfo, blockCount *int, depth int) {
	if depth > 10 {
		return
	}
	for _, child := range node.Children {
		if *blockCount >= 1000 {
			return
		}
		e.extractBlocksFromNodeOptimized(child, filepath, context, blocks, blockCount)
	}
}

func (e *ExactDuplicatedCodeAnalyzer) isFunctionDefinition(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || node.Children[0].Value == "defn-" ||
			node.Children[0].Value == "defmacro" || node.Children[0].Value == "defmethod")
}

func (e *ExactDuplicatedCodeAnalyzer) isLetBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "let" || node.Children[0].Value == "when-let" ||
			node.Children[0].Value == "if-let" || node.Children[0].Value == "binding")
}

func (e *ExactDuplicatedCodeAnalyzer) isConditionalBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "if" || node.Children[0].Value == "when" ||
			node.Children[0].Value == "cond" || node.Children[0].Value == "case" ||
			node.Children[0].Value == "condp")
}

func (e *ExactDuplicatedCodeAnalyzer) isLoopBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "loop" || node.Children[0].Value == "doseq" ||
			node.Children[0].Value == "dotimes" || node.Children[0].Value == "for")
}

func (e *ExactDuplicatedCodeAnalyzer) isSignificantBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) >= 4
}

func (e *ExactDuplicatedCodeAnalyzer) extractFunctionBlockOptimized(node *reader.RichNode, filepath string, blocks *[]ExactCodeBlockInfo, blockCount *int) bool {
	context := ""
	if len(node.Children) > 1 {
		context = node.Children[1].Value
	}
	return e.extractTypedBlockOptimized(node, filepath, "function", context, blocks, blockCount)
}

func (e *ExactDuplicatedCodeAnalyzer) extractLetBlockOptimized(node *reader.RichNode, filepath, context string, blocks *[]ExactCodeBlockInfo, blockCount *int) bool {
	return e.extractTypedBlockOptimized(node, filepath, "let-block", context, blocks, blockCount)
}

func (e *ExactDuplicatedCodeAnalyzer) extractConditionalBlockOptimized(node *reader.RichNode, filepath, context string, blocks *[]ExactCodeBlockInfo, blockCount *int) bool {
	return e.extractTypedBlockOptimized(node, filepath, "conditional-block", context, blocks, blockCount)
}

func (e *ExactDuplicatedCodeAnalyzer) extractLoopBlockOptimized(node *reader.RichNode, filepath, context string, blocks *[]ExactCodeBlockInfo, blockCount *int) bool {
	return e.extractTypedBlockOptimized(node, filepath, "loop-block", context, blocks, blockCount)
}

func (e *ExactDuplicatedCodeAnalyzer) extractGenericBlockOptimized(node *reader.RichNode, filepath, context string, blocks *[]ExactCodeBlockInfo, blockCount *int) bool {
	return e.extractTypedBlockOptimized(node, filepath, "generic-block", context, blocks, blockCount)
}

func (e *ExactDuplicatedCodeAnalyzer) extractTypedBlockOptimized(node *reader.RichNode, filepath, blockType, context string, blocks *[]ExactCodeBlockInfo, blockCount *int) bool {
	lines := e.calculateLinesOptimized(node)
	tokens := e.calculateTokensOptimized(node)

	if lines < e.minLines || tokens < e.minTokens {
		return false
	}

	normalizedAST := e.normalizeASTOptimized(node)
	if e.isTrivialPattern(normalizedAST) {
		return false
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(normalizedAST)))

	blockInfo := ExactCodeBlockInfo{
		Hash:      hash,
		File:      filepath,
		Location:  node.Location,
		Lines:     lines,
		Tokens:    tokens,
		BlockType: blockType,
		Context:   context,
	}

	*blocks = append(*blocks, blockInfo)
	*blockCount++
	return true
}

func (e *ExactDuplicatedCodeAnalyzer) normalizeASTOptimized(node *reader.RichNode) string {

	nodeKey := fmt.Sprintf("%p", node)
	e.cacheMutex.RLock()
	cached, found := e.normalizedCache[nodeKey]
	e.cacheMutex.RUnlock()
	if found {
		e.cacheHits++
		return cached
	}
	e.cacheMisses++

	var builder strings.Builder
	var visit func(*reader.RichNode)
	visit = func(n *reader.RichNode) {
		if n == nil {
			return
		}
		switch n.Type {
		case reader.NodeList, reader.NodeVector, reader.NodeMap, reader.NodeSet:
			builder.WriteString("(")
			for _, child := range n.Children {
				visit(child)
				builder.WriteString(" ")
			}
			builder.WriteString(")")
		case reader.NodeSymbol:
			builder.WriteString(e.normalizeSymbol(n.Value))
		case reader.NodeKeyword:
			if e.isImportantKeyword(n.Value) {
				builder.WriteString(n.Value)
			} else {
				builder.WriteString(":KEYWORD")
			}
		case reader.NodeString:
			builder.WriteString("\"STRING\"")
		case reader.NodeNumber:
			builder.WriteString("NUMBER")
		case reader.NodeBool:
			builder.WriteString(n.Value)
		default:

		}
	}
	visit(node)
	normalized := builder.String()

	e.cacheMutex.Lock()
	if len(e.normalizedCache) < e.maxCacheSize {
		e.normalizedCache[nodeKey] = normalized
	}
	e.cacheMutex.Unlock()

	return normalized
}

func (e *ExactDuplicatedCodeAnalyzer) normalizeSymbol(symbol string) string {
	if e.isNumericLiteral(symbol) {
		return "NUMBER"
	}
	for pattern, replacement := range exactSymbolReplacements {
		if exactSymbolPatterns[pattern].MatchString(symbol) {
			return replacement
		}
	}
	return "VAR"
}

func (e *ExactDuplicatedCodeAnalyzer) isImportantKeyword(keyword string) bool {

	return strings.HasPrefix(keyword, ":")
}

func (e *ExactDuplicatedCodeAnalyzer) isNumericLiteral(value string) bool {
	return exactNumericLiteralRegex.MatchString(value)
}

func (e *ExactDuplicatedCodeAnalyzer) calculateLinesOptimized(node *reader.RichNode) int {
	if node.Location == nil {
		return 0
	}
	return node.Location.EndLine - node.Location.StartLine + 1
}

func (e *ExactDuplicatedCodeAnalyzer) calculateTokensOptimized(node *reader.RichNode) int {
	count := 0
	var visit func(*reader.RichNode)
	visit = func(n *reader.RichNode) {
		if n != nil {
			count++
			for _, child := range n.Children {
				visit(child)
			}
		}
	}
	visit(node)
	return count
}

func (e *ExactDuplicatedCodeAnalyzer) isTrivialPattern(normalizedAST string) bool {
	trivialPatterns := []string{
		`\(\(VAR VAR\) \(\)\)`,
		`\(\(VAR\) \(\)\)`,
	}
	for _, p := range trivialPatterns {
		match, _ := regexp.MatchString(p, normalizedAST)
		if match {
			return true
		}
	}
	return false
}

func (e *ExactDuplicatedCodeAnalyzer) createExactDuplicationMessage(block ExactCodeBlockInfo, count int) string {
	return fmt.Sprintf("Bloco de código duplicado encontrado. Hash: %s. Esta é a %dª ocorrência.",
		block.Hash, count)
}

func (e *ExactDuplicatedCodeAnalyzer) cleanupIfNeeded() {
	if time.Since(e.lastCleanup) > e.cleanupInterval {
		if len(e.codeBlocks) > e.maxCacheSize*2 {
			e.codeBlocks = make(map[string][]ExactCodeBlockInfo)
		}
		e.normalizedCache = make(map[string]string)
		e.lastCleanup = time.Now()
	}
}

func (e *ExactDuplicatedCodeAnalyzer) Reset() {
	e.codeBlocks = make(map[string][]ExactCodeBlockInfo)
	e.totalBlocksProcessed = 0
	e.cacheHits = 0
	e.cacheMisses = 0
}

func (e *ExactDuplicatedCodeAnalyzer) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"DuplicatedBlocks":     e.codeBlocks,
		"TotalBlocksProcessed": e.totalBlocksProcessed,
		"CacheHits":            e.cacheHits,
		"CacheMisses":          e.cacheMisses,
		"NormalizedCacheSize":  len(e.normalizedCache),
		"CodeBlocksHashes":     len(e.codeBlocks),
	}
}

func (e *ExactDuplicatedCodeAnalyzer) SetThresholds(minLines, minTokens int) {
	e.minLines = minLines
	e.minTokens = minTokens
}

type DuplicatedCodeExactRule struct {
	Rule
}

func (r *DuplicatedCodeExactRule) Meta() Rule {
	return r.Rule
}

func (r *DuplicatedCodeExactRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	return nil
}

func init() {
	rule := &DuplicatedCodeExactRule{
		Rule: Rule{
			ID:          "duplicated-code-exact",
			Name:        "Duplicated Code (Exact)",
			Description: "Finds blocks of code that are exactly identical after normalization.",
			Severity:    SeverityWarning,
		},
	}
	RegisterRule(rule)
}
