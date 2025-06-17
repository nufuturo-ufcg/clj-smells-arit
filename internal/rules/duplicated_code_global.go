package rules

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/cespare/goclj/parse"
	"github.com/thlaurentino/arit/internal/reader"
)

type GlobalDuplicatedCodeAnalyzer struct {
	mu          sync.Mutex
	codeBlocks  map[string][]CodeBlockInfo
	minLines    int
	minTokens   int
	initialized bool

	compiledRegexes map[string]*regexp.Regexp
	regexOnce       sync.Once
}

type CodeBlockInfo struct {
	Hash          string
	Content       string
	NormalizedAST string
	File          string
	Location      *reader.Location
	Lines         int
	Tokens        int
	BlockType     string
	Context       string
}

var globalAnalyzer *GlobalDuplicatedCodeAnalyzer

var (
	numericLiteralRegex *regexp.Regexp
	symbolPatterns      map[string]*regexp.Regexp
	symbolReplacements  map[string]string
	regexInitOnce       sync.Once
)

// Regex pré-compiled patterns for symbol normalization
func initRegexPatterns() {
	regexInitOnce.Do(func() {

		numericLiteralRegex = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)

		symbolPatterns = make(map[string]*regexp.Regexp)
		symbolReplacements = make(map[string]string)

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
			{`^(?:handle|manage|execute)-`, "PROCESS_FUNC"},
			{`^(?:fetch|retrieve|load)-`, "GET_FUNC"},
			{`^(?:save|store|persist|update)-`, "SET_FUNC"},
			{`^(?:check|verify|ensure)-`, "VALIDATE_FUNC"},
			{`^(?:compute|determine|find)-`, "CALC_FUNC"},
			{`^(?:make|build|generate)-`, "CREATE_FUNC"},
			{`^(?:parse|format|transform|convert)-`, "TRANSFORM_FUNC"},
		}

		for _, p := range patterns {

			optimizedPattern := strings.ReplaceAll(p.pattern, "(?:", "(?>")
			compiled := regexp.MustCompile(optimizedPattern)
			symbolPatterns[p.pattern] = compiled
			symbolReplacements[p.pattern] = p.replacement
		}
	})
}

func GetGlobalDuplicatedCodeAnalyzer() *GlobalDuplicatedCodeAnalyzer {
	if globalAnalyzer == nil {

		initRegexPatterns()

		globalAnalyzer = &GlobalDuplicatedCodeAnalyzer{
			codeBlocks:      make(map[string][]CodeBlockInfo),
			compiledRegexes: make(map[string]*regexp.Regexp),
			minLines:        3,
			minTokens:       15,
		}
	}
	return globalAnalyzer
}

func (g *GlobalDuplicatedCodeAnalyzer) AnalyzeTree(tree *parse.Tree, richNodes []*reader.RichNode, filepath string) []Finding {
	g.mu.Lock()
	defer g.mu.Unlock()

	var findings []Finding

	blocks := g.extractAllCodeBlocks(richNodes, filepath)

	for _, block := range blocks {
		g.codeBlocks[block.Hash] = append(g.codeBlocks[block.Hash], block)

		if len(g.codeBlocks[block.Hash]) > 1 {
			finding := Finding{
				RuleID:   "duplicated-code-global",
				Message:  g.createDuplicationMessage(block, len(g.codeBlocks[block.Hash])),
				Filepath: filepath,
				Location: block.Location,
				Severity: SeverityWarning,
			}
			findings = append(findings, finding)
		}
	}

	return findings
}

func (g *GlobalDuplicatedCodeAnalyzer) extractAllCodeBlocks(nodes []*reader.RichNode, filepath string) []CodeBlockInfo {
	var blocks []CodeBlockInfo

	for _, node := range nodes {
		g.extractBlocksFromNode(node, filepath, "", &blocks)
	}
	return blocks
}

func (g *GlobalDuplicatedCodeAnalyzer) extractBlocksFromNode(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if node == nil {
		return
	}

	switch {
	case g.isFunctionDefinition(node):
		g.extractFunctionBlock(node, filepath, blocks)
	case g.isLetBlock(node):
		g.extractLetBlock(node, filepath, context, blocks)
	case g.isConditionalBlock(node):
		g.extractConditionalBlock(node, filepath, context, blocks)
	case g.isLoopBlock(node):
		g.extractLoopBlock(node, filepath, context, blocks)
	case g.isSignificantBlock(node):
		g.extractGenericBlock(node, filepath, context, blocks)
	}

	newContext := context
	if g.isFunctionDefinition(node) && len(node.Children) > 1 {
		newContext = node.Children[1].Value
	}

	for _, child := range node.Children {
		g.extractBlocksFromNode(child, filepath, newContext, blocks)
	}
}

func (g *GlobalDuplicatedCodeAnalyzer) isFunctionDefinition(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" ||
			node.Children[0].Value == "defn-" ||
			node.Children[0].Value == "defmacro" ||
			node.Children[0].Value == "defmethod")
}

func (g *GlobalDuplicatedCodeAnalyzer) isLetBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "let" ||
			node.Children[0].Value == "when-let" ||
			node.Children[0].Value == "if-let" ||
			node.Children[0].Value == "binding")
}

func (g *GlobalDuplicatedCodeAnalyzer) isConditionalBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "if" ||
			node.Children[0].Value == "when" ||
			node.Children[0].Value == "cond" ||
			node.Children[0].Value == "case" ||
			node.Children[0].Value == "condp")
}

func (g *GlobalDuplicatedCodeAnalyzer) isLoopBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "loop" ||
			node.Children[0].Value == "doseq" ||
			node.Children[0].Value == "dotimes" ||
			node.Children[0].Value == "for")
}

func (g *GlobalDuplicatedCodeAnalyzer) isSignificantBlock(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	complexity := g.calculateComplexity(node)
	return complexity >= g.minTokens
}

func (g *GlobalDuplicatedCodeAnalyzer) calculateComplexity(node *reader.RichNode) int {
	if node == nil {
		return 0
	}

	complexity := 1
	for _, child := range node.Children {
		complexity += g.calculateComplexity(child)
	}

	return complexity
}

func (g *GlobalDuplicatedCodeAnalyzer) extractFunctionBlock(node *reader.RichNode, filepath string, blocks *[]CodeBlockInfo) {
	g.extractTypedBlock(node, filepath, "function", "", blocks)
}

func (g *GlobalDuplicatedCodeAnalyzer) extractLetBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if len(node.Children) >= 3 {
		g.extractTypedBlock(node, filepath, "let-block", context, blocks)
	}
}

func (g *GlobalDuplicatedCodeAnalyzer) extractConditionalBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if len(node.Children) >= 2 {
		g.extractTypedBlock(node, filepath, "conditional-block", context, blocks)
	}
}

func (g *GlobalDuplicatedCodeAnalyzer) extractLoopBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if len(node.Children) >= 2 {
		g.extractTypedBlock(node, filepath, "loop-block", context, blocks)
	}
}

func (g *GlobalDuplicatedCodeAnalyzer) extractGenericBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	g.extractTypedBlock(node, filepath, "generic-block", context, blocks)
}

func (g *GlobalDuplicatedCodeAnalyzer) extractTypedBlock(node *reader.RichNode, filepath, blockType, context string, blocks *[]CodeBlockInfo) {
	content := g.extractNodeContent(node)
	normalizedAST := g.normalizeAST(node)

	lines := g.calculateLines(node)
	tokens := g.calculateTokens(node)

	if lines >= g.minLines && tokens >= g.minTokens {
		hash := fmt.Sprintf("%x", md5.Sum([]byte(normalizedAST)))

		block := CodeBlockInfo{
			Hash:          hash,
			Content:       content,
			NormalizedAST: normalizedAST,
			File:          filepath,
			Location:      node.Location,
			Lines:         lines,
			Tokens:        tokens,
			BlockType:     blockType,
			Context:       context,
		}

		*blocks = append(*blocks, block)
	}
}

func (g *GlobalDuplicatedCodeAnalyzer) normalizeAST(node *reader.RichNode) string {
	if node == nil {
		return ""
	}

	var parts []string

	switch node.Type {
	case reader.NodeSymbol:

		normalized := g.normalizeSymbol(node.Value)
		parts = append(parts, normalized)

	case reader.NodeString:
		parts = append(parts, "STRING_LITERAL")

	case reader.NodeKeyword:

		if g.isImportantKeyword(node.Value) {
			parts = append(parts, ":"+node.Value)
		} else {
			parts = append(parts, ":KEYWORD")
		}

	case reader.NodeList:
		parts = append(parts, "(")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, g.normalizeAST(child))
		}
		parts = append(parts, ")")

	case reader.NodeVector:
		parts = append(parts, "[")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, g.normalizeAST(child))
		}
		parts = append(parts, "]")

	case reader.NodeMap:
		parts = append(parts, "{")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, g.normalizeAST(child))
		}
		parts = append(parts, "}")

	default:
		if node.Value != "" {

			if g.isNumericLiteral(node.Value) {
				parts = append(parts, "NUMBER")
			} else {
				parts = append(parts, g.normalizeSymbol(node.Value))
			}
		}
	}

	return strings.Join(parts, "")
}

var coreFunctionSet = map[string]bool{
	"map": true, "filter": true, "reduce": true, "apply": true, "partial": true, "comp": true,
	"let": true, "when": true, "if": true, "cond": true, "case": true, "defn": true, "def": true,
	"assoc": true, "dissoc": true, "get": true, "get-in": true, "update": true, "update-in": true,
	"first": true, "rest": true, "last": true, "count": true, "empty?": true, "seq": true,
	"+": true, "-": true, "*": true, "/": true, "=": true, "<": true, ">": true, "<=": true, ">=": true, "not=": true,
}

func (g *GlobalDuplicatedCodeAnalyzer) normalizeSymbol(symbol string) string {

	if coreFunctionSet[symbol] {
		return symbol
	}

	if len(symbol) == 0 {
		return "VAR"
	}

	firstChar := symbol[0]
	switch {
	case firstChar >= 'a' && firstChar <= 'z':

	case firstChar >= 'A' && firstChar <= 'Z':

		return "VAR"
	default:

		return "VAR"
	}

	for pattern, compiled := range symbolPatterns {
		if compiled.MatchString(symbol) {
			return symbolReplacements[pattern]
		}
	}

	return "VAR"
}

var importantKeywordSet = map[string]bool{
	"require": true, "import": true, "refer": true, "as": true, "exclude": true, "only": true,
	"keys": true, "vals": true, "strs": true, "syms": true,
}

func (g *GlobalDuplicatedCodeAnalyzer) isImportantKeyword(keyword string) bool {
	return importantKeywordSet[keyword]
}

func (g *GlobalDuplicatedCodeAnalyzer) isNumericLiteral(value string) bool {

	if len(value) == 0 {
		return false
	}

	for i, r := range value {
		switch {
		case r >= '0' && r <= '9':
			continue
		case r == '-' && i == 0:
			continue
		case r == '.' && i > 0:
			continue
		default:
			return false
		}
	}

	return numericLiteralRegex.MatchString(value)
}

func (g *GlobalDuplicatedCodeAnalyzer) calculateLines(node *reader.RichNode) int {
	if node.Location != nil && node.Location.EndLine > node.Location.StartLine {
		return node.Location.EndLine - node.Location.StartLine + 1
	}

	complexity := g.calculateComplexity(node)
	return max(1, complexity/5)
}

func (g *GlobalDuplicatedCodeAnalyzer) calculateTokens(node *reader.RichNode) int {
	return g.calculateComplexity(node)
}

func (g *GlobalDuplicatedCodeAnalyzer) extractNodeContent(node *reader.RichNode) string {
	if node == nil {
		return ""
	}

	var parts []string

	switch node.Type {
	case reader.NodeSymbol, reader.NodeKeyword, reader.NodeString:
		parts = append(parts, node.Value)

	case reader.NodeList:
		parts = append(parts, "(")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, g.extractNodeContent(child))
		}
		parts = append(parts, ")")

	case reader.NodeVector:
		parts = append(parts, "[")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, g.extractNodeContent(child))
		}
		parts = append(parts, "]")

	case reader.NodeMap:
		parts = append(parts, "{")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, g.extractNodeContent(child))
		}
		parts = append(parts, "}")

	default:
		if node.Value != "" {
			parts = append(parts, node.Value)
		}
	}

	return strings.Join(parts, "")
}

func (g *GlobalDuplicatedCodeAnalyzer) createDuplicationMessage(block CodeBlockInfo, count int) string {
	var otherFiles []string
	for _, otherBlock := range g.codeBlocks[block.Hash] {
		if otherBlock.File != block.File {
			context := otherBlock.Context
			if context == "" {
				context = "unknown"
			}
			otherFiles = append(otherFiles, fmt.Sprintf("%s:%s", otherBlock.File, context))
		}
	}

	blockTypeDesc := map[string]string{
		"function":          "function",
		"let-block":         "let block",
		"conditional-block": "conditional block",
		"loop-block":        "loop block",
		"generic-block":     "code block",
	}

	desc := blockTypeDesc[block.BlockType]
	if desc == "" {
		desc = "code block"
	}

	context := block.Context
	if context == "" {
		context = "unknown"
	}

	message := fmt.Sprintf("Duplicated %s detected in %q (%d occurrences, %d lines, %d tokens)",
		desc, context, count, block.Lines, block.Tokens)

	if len(otherFiles) > 0 {
		message += fmt.Sprintf(". Also found in: %s", strings.Join(otherFiles, ", "))
	}

	return message
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (g *GlobalDuplicatedCodeAnalyzer) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.codeBlocks = make(map[string][]CodeBlockInfo)
}

func (g *GlobalDuplicatedCodeAnalyzer) GetStatistics() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()

	totalBlocks := 0
	duplicatedHashes := 0
	duplicatedBlocks := 0

	for _, blocks := range g.codeBlocks {
		totalBlocks += len(blocks)
		if len(blocks) > 1 {
			duplicatedHashes++
			duplicatedBlocks += len(blocks)
		}
	}

	blocksByType := make(map[string]int)
	for _, blocks := range g.codeBlocks {
		for _, block := range blocks {
			blocksByType[block.BlockType]++
		}
	}

	return map[string]interface{}{
		"total_blocks":      totalBlocks,
		"unique_hashes":     len(g.codeBlocks),
		"duplicated_hashes": duplicatedHashes,
		"duplicated_blocks": duplicatedBlocks,
		"blocks_by_type":    blocksByType,
		"duplication_ratio": float64(duplicatedBlocks) / float64(totalBlocks),
	}
}

func (g *GlobalDuplicatedCodeAnalyzer) SetThresholds(minLines, minTokens int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.minLines = minLines
	g.minTokens = minTokens
}

type DuplicatedCodeGlobalRule struct {
	Rule
}

func (r *DuplicatedCodeGlobalRule) Meta() Rule {
	return r.Rule
}

func (r *DuplicatedCodeGlobalRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	return nil
}

func init() {
	defaultRule := &DuplicatedCodeGlobalRule{
		Rule: Rule{
			ID:   "duplicated-code-global",
			Name: "Duplicated Code (Global Analysis)",
			Description: "Detects duplicated code blocks across the entire codebase using AST-based structural analysis. " +
				"This rule identifies similar code patterns in functions, let blocks, conditionals, loops, and other significant code structures, " +
				"normalizing variable names and literals to focus on structural similarity rather than exact text matches.",
			Severity: SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
}
