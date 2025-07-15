package rules

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/cespare/goclj/parse"
	"github.com/thlaurentino/arit/internal/reader"
)

type DuplicatedCodeAnalyzer struct {
	mu                   sync.Mutex
	exactCodeBlocks      map[string][]CodeBlockInfo
	processedNodes       map[*reader.RichNode]string
	nodeCounter          int64
	totalBlocksProcessed int64

	normalizationCache map[string]string
	cacheHits          int64
	cacheMisses        int64

	exactMinLines    int
	exactMinTokens   int
	maxCacheSize     int
	maxBlocksPerFile int

	semanticMapNormalization bool

	detectFunctions  bool
	detectCodeBlocks bool
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
	NodeID        int64
}

var (
	unifiedAnalyzer     *DuplicatedCodeAnalyzer
	analyzerInitOnce    sync.Once
	numericLiteralRegex *regexp.Regexp
	symbolPatterns      map[string]*regexp.Regexp
	symbolReplacements  map[string]string
	regexInitOnce       sync.Once
)

func initUnifiedRegexPatterns() {
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
			compiled := regexp.MustCompile(p.pattern)
			symbolPatterns[p.pattern] = compiled
			symbolReplacements[p.pattern] = p.replacement
		}
	})
}

func GetDuplicatedCodeAnalyzer() *DuplicatedCodeAnalyzer {
	analyzerInitOnce.Do(func() {
		initUnifiedRegexPatterns()
		unifiedAnalyzer = &DuplicatedCodeAnalyzer{
			exactCodeBlocks:    make(map[string][]CodeBlockInfo),
			processedNodes:     make(map[*reader.RichNode]string),
			normalizationCache: make(map[string]string),

			exactMinLines:    1,
			exactMinTokens:   5,
			maxCacheSize:     10000,
			maxBlocksPerFile: 1000,

			semanticMapNormalization: true,

			detectFunctions:  true,
			detectCodeBlocks: false,
		}
	})
	return unifiedAnalyzer
}

func (d *DuplicatedCodeAnalyzer) SetDetectionMode(functions, codeBlocks bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.detectFunctions = functions
	d.detectCodeBlocks = codeBlocks
}

func (d *DuplicatedCodeAnalyzer) DetectOnlyFunctions() {
	d.SetDetectionMode(true, false)
}

func (d *DuplicatedCodeAnalyzer) DetectOnlyCodeBlocks() {
	d.SetDetectionMode(false, true)
}

func (d *DuplicatedCodeAnalyzer) DetectBoth() {
	d.SetDetectionMode(true, true)
}

func (d *DuplicatedCodeAnalyzer) GetDetectionMode() (functions, codeBlocks bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.detectFunctions, d.detectCodeBlocks
}

func (d *DuplicatedCodeAnalyzer) AnalyzeTree(_ *parse.Tree, richNodes []*reader.RichNode, filepath string) []Finding {
	d.mu.Lock()
	defer d.mu.Unlock()

	var findings []Finding

	blocks := d.extractAllCodeBlocks(richNodes, filepath)

	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].File != blocks[j].File {
			return blocks[i].File < blocks[j].File
		}
		if blocks[i].Location != nil && blocks[j].Location != nil {
			if blocks[i].Location.StartLine != blocks[j].Location.StartLine {
				return blocks[i].Location.StartLine < blocks[j].Location.StartLine
			}
			return blocks[i].Location.StartColumn < blocks[j].Location.StartColumn
		}
		return blocks[i].NodeID < blocks[j].NodeID
	})

	for _, block := range blocks {
		exactFindings := d.processExactDuplicate(block, filepath)
		findings = append(findings, exactFindings...)
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Filepath != findings[j].Filepath {
			return findings[i].Filepath < findings[j].Filepath
		}
		if findings[i].Location != nil && findings[j].Location != nil {
			if findings[i].Location.StartLine != findings[j].Location.StartLine {
				return findings[i].Location.StartLine < findings[j].Location.StartLine
			}
			return findings[i].Location.StartColumn < findings[j].Location.StartColumn
		}
		return findings[i].RuleID < findings[j].RuleID
	})

	return findings
}

func (d *DuplicatedCodeAnalyzer) extractAllCodeBlocks(nodes []*reader.RichNode, filepath string) []CodeBlockInfo {
	var blocks []CodeBlockInfo
	blockCount := 0

	for _, node := range nodes {
		if blockCount >= d.maxBlocksPerFile {
			break
		}
		d.extractBlocksFromNode(node, filepath, "", &blocks, &blockCount)
	}
	return blocks
}

func (d *DuplicatedCodeAnalyzer) extractBlocksFromNode(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo, blockCount *int) {
	if node == nil || *blockCount >= d.maxBlocksPerFile {
		return
	}

	switch {
	case d.isFunctionDefinition(node):
		if d.detectFunctions {
			d.extractTypedBlock(node, filepath, "function", context, blocks, blockCount)
		}
	case d.isLetBlock(node):
		if d.detectCodeBlocks {
			d.extractTypedBlock(node, filepath, "let-block", context, blocks, blockCount)
		}
	case d.isConditionalBlock(node):
		if d.detectCodeBlocks {
			d.extractTypedBlock(node, filepath, "conditional-block", context, blocks, blockCount)
		}
	case d.isLoopBlock(node):
		if d.detectCodeBlocks {
			d.extractTypedBlock(node, filepath, "loop-block", context, blocks, blockCount)
		}
	case d.isSignificantBlock(node):

		if d.detectCodeBlocks {
			d.extractTypedBlock(node, filepath, "generic-block", context, blocks, blockCount)
		}
	}

	newContext := context
	if d.isFunctionDefinition(node) && len(node.Children) > 1 {
		newContext = node.Children[1].Value
	}

	for _, child := range node.Children {
		if *blockCount >= d.maxBlocksPerFile {
			break
		}
		d.extractBlocksFromNode(child, filepath, newContext, blocks, blockCount)
	}
}

func fastHash(data string) string {
	h := fnv.New64a()
	_, err := h.Write([]byte(data))
	if err != nil {

		return fmt.Sprintf("%x", len(data))
	}
	return fmt.Sprintf("%x", h.Sum64())
}

func (d *DuplicatedCodeAnalyzer) extractTypedBlock(node *reader.RichNode, filepath, blockType, context string, blocks *[]CodeBlockInfo, blockCount *int) {
	d.nodeCounter++
	nodeID := d.nodeCounter

	content := d.extractNodeContent(node)
	lines := d.calculateLines(node)
	tokens := d.calculateTokens(node)

	exactMeetsThreshold := lines >= d.exactMinLines && tokens >= d.exactMinTokens

	if !exactMeetsThreshold {
		return
	}

	exactNormalized := d.normalizeAST(node)
	exactHash := fastHash("exact:" + exactNormalized)

	block := CodeBlockInfo{
		Hash:          exactHash,
		Content:       content,
		NormalizedAST: exactNormalized,
		File:          filepath,
		Location:      node.Location,
		Lines:         lines,
		Tokens:        tokens,
		BlockType:     blockType,
		Context:       context,
		NodeID:        nodeID,
	}
	*blocks = append(*blocks, block)

	*blockCount++
	d.totalBlocksProcessed++
}

type mapPair struct {
	key       *reader.RichNode
	value     *reader.RichNode
	keyString string
}

func (d *DuplicatedCodeAnalyzer) normalizeAST(node *reader.RichNode) string {
	if node == nil {
		return ""
	}

	var builder strings.Builder
	var visit func(*reader.RichNode)

	visit = func(n *reader.RichNode) {
		if n == nil {
			return
		}

		switch n.Type {
		case reader.NodeList, reader.NodeVector, reader.NodeSet:
			builder.WriteString("(")
			for _, child := range n.Children {
				visit(child)
				builder.WriteString(" ")
			}
			builder.WriteString(")")
		case reader.NodeMap:
			if d.semanticMapNormalization {

				builder.WriteString("{")
				pairs := d.extractAndSortMapPairs(n.Children)
				for i, pair := range pairs {
					if i > 0 {
						builder.WriteString(" ")
					}
					visit(pair.key)
					builder.WriteString(" ")
					visit(pair.value)
				}
				builder.WriteString("}")
			} else {

				builder.WriteString("{")
				for _, child := range n.Children {
					visit(child)
					builder.WriteString(" ")
				}
				builder.WriteString("}")
			}
		case reader.NodeSymbol:
			builder.WriteString(d.normalizeSymbol(n.Value))
		case reader.NodeKeyword:
			if d.isImportantKeyword(n.Value) {
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
	return builder.String()
}

func (d *DuplicatedCodeAnalyzer) extractAndSortMapPairs(children []*reader.RichNode) []mapPair {
	var pairs []mapPair

	for i := 0; i < len(children); i += 2 {
		if i+1 < len(children) {
			key := children[i]
			value := children[i+1]

			keyString := ""
			if key.Type == reader.NodeKeyword {
				keyString = key.Value
			} else {
				keyString = d.extractNodeContent(key)
			}

			pairs = append(pairs, mapPair{
				key:       key,
				value:     value,
				keyString: keyString,
			})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].keyString < pairs[j].keyString
	})

	return pairs
}

func (d *DuplicatedCodeAnalyzer) normalizeSymbol(symbol string) string {
	if d.isCoreFunctionSymbol(symbol) {
		return symbol
	}
	if d.isNumericLiteral(symbol) {
		return "NUMBER"
	}

	return "VAR"
}

func (d *DuplicatedCodeAnalyzer) isFunctionDefinition(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" ||
			node.Children[0].Value == "defn-" ||
			node.Children[0].Value == "defmacro" ||
			node.Children[0].Value == "defmethod")
}

func (d *DuplicatedCodeAnalyzer) isLetBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "let" ||
			node.Children[0].Value == "when-let" ||
			node.Children[0].Value == "if-let" ||
			node.Children[0].Value == "binding")
}

func (d *DuplicatedCodeAnalyzer) isConditionalBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "if" ||
			node.Children[0].Value == "when" ||
			node.Children[0].Value == "cond" ||
			node.Children[0].Value == "case" ||
			node.Children[0].Value == "condp")
}

func (d *DuplicatedCodeAnalyzer) isLoopBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "loop" ||
			node.Children[0].Value == "doseq" ||
			node.Children[0].Value == "dotimes" ||
			node.Children[0].Value == "for")
}

func (d *DuplicatedCodeAnalyzer) isSignificantBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) >= 4
}

func (d *DuplicatedCodeAnalyzer) processExactDuplicate(block CodeBlockInfo, filepath string) []Finding {
	var findings []Finding
	d.exactCodeBlocks[block.Hash] = append(d.exactCodeBlocks[block.Hash], block)

	if len(d.exactCodeBlocks[block.Hash]) > 1 {

		finding := Finding{
			RuleID:   "duplicated-code",
			Message:  d.createDuplicationMessage(block, len(d.exactCodeBlocks[block.Hash])),
			Filepath: filepath,
			Location: block.Location,
			Severity: SeverityWarning,
		}
		findings = append(findings, finding)
	} else {

	}

	return findings
}

func (d *DuplicatedCodeAnalyzer) createDuplicationMessage(block CodeBlockInfo, count int) string {
	var otherFiles []string
	blocks := d.exactCodeBlocks[block.Hash]

	for _, otherBlock := range blocks {
		if otherBlock.File != block.File {
			context := otherBlock.Context
			if context == "" {
				context = "unknown"
			}
			otherFiles = append(otherFiles, fmt.Sprintf("%s:%s", otherBlock.File, context))
		}
	}
	sort.Strings(otherFiles)

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

	titleCase := func(s string) string {
		if len(s) == 0 {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}

	message := fmt.Sprintf("%s %s detected in %q (%d occurrences, %d lines, %d tokens)",
		titleCase("duplicated"), desc, context, count, block.Lines, block.Tokens)

	if len(otherFiles) > 0 {
		message += fmt.Sprintf(". Also found in: %s", strings.Join(otherFiles, ", "))
	}

	return message
}

func (d *DuplicatedCodeAnalyzer) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.exactCodeBlocks = make(map[string][]CodeBlockInfo)
	d.processedNodes = make(map[*reader.RichNode]string)
	d.normalizationCache = make(map[string]string)

	d.nodeCounter = 0
	d.totalBlocksProcessed = 0
	d.cacheHits = 0
	d.cacheMisses = 0
}

func (d *DuplicatedCodeAnalyzer) GetStatistics() map[string]interface{} {
	d.mu.Lock()
	defer d.mu.Unlock()

	totalExactBlocks := 0
	duplicatedExactHashes := 0
	for _, blocks := range d.exactCodeBlocks {
		totalExactBlocks += len(blocks)
		if len(blocks) > 1 {
			duplicatedExactHashes++
		}
	}

	return map[string]interface{}{
		"total_exact_blocks":       totalExactBlocks,
		"duplicated_exact_hashes":  duplicatedExactHashes,
		"total_blocks_processed":   d.totalBlocksProcessed,
		"normalization_cache_hits": d.cacheHits,
		"normalization_cache_miss": d.cacheMisses,
		"cache_size":               len(d.normalizationCache),
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type DuplicatedCodeExactRule struct {
	Rule
	DetectFunctions          bool `json:"detect_functions" yaml:"detect_functions"`
	DetectCodeBlocks         bool `json:"detect_code_blocks" yaml:"detect_code_blocks"`
	SemanticMapNormalization bool `json:"semantic_map_normalization" yaml:"semantic_map_normalization"`
	ExactMinLines            int  `json:"exact_min_lines" yaml:"exact_min_lines"`
	ExactMinTokens           int  `json:"exact_min_tokens" yaml:"exact_min_tokens"`
	MaxBlocksPerFile         int  `json:"max_blocks_per_file" yaml:"max_blocks_per_file"`
}

func (r *DuplicatedCodeExactRule) Meta() Rule {
	return r.Rule
}

func (r *DuplicatedCodeExactRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {

	globalAnalyzer := GetDuplicatedCodeAnalyzer()
	globalAnalyzer.SetDetectionMode(r.DetectFunctions, r.DetectCodeBlocks)
	globalAnalyzer.semanticMapNormalization = r.SemanticMapNormalization
	globalAnalyzer.exactMinLines = r.ExactMinLines
	globalAnalyzer.exactMinTokens = r.ExactMinTokens
	globalAnalyzer.maxBlocksPerFile = r.MaxBlocksPerFile

	return nil
}

func (r *DuplicatedCodeExactRule) getConfiguredAnalyzer() *DuplicatedCodeAnalyzer {
	analyzer := &DuplicatedCodeAnalyzer{
		exactCodeBlocks:    make(map[string][]CodeBlockInfo),
		processedNodes:     make(map[*reader.RichNode]string),
		normalizationCache: make(map[string]string),

		exactMinLines:            r.ExactMinLines,
		exactMinTokens:           r.ExactMinTokens,
		maxBlocksPerFile:         r.MaxBlocksPerFile,
		semanticMapNormalization: r.SemanticMapNormalization,
		detectFunctions:          r.DetectFunctions,
		detectCodeBlocks:         r.DetectCodeBlocks,
	}

	return analyzer
}

func init() {

	RegisterRule(&DuplicatedCodeExactRule{
		Rule: Rule{
			ID:          "duplicated-code",
			Name:        "Exact Code Duplication",
			Description: "Detects exact duplicated code blocks that are identical in structure and content",
			Severity:    SeverityWarning,
		},

		DetectFunctions:          true,
		DetectCodeBlocks:         false,
		SemanticMapNormalization: true,
		ExactMinLines:            1,
		ExactMinTokens:           5,
		MaxBlocksPerFile:         1000,
	})
}

func (d *DuplicatedCodeAnalyzer) isCoreFunctionSymbol(symbol string) bool {
	coreFunctions := map[string]bool{
		"map": true, "filter": true, "reduce": true, "apply": true, "partial": true, "comp": true,
		"let": true, "when": true, "if": true, "cond": true, "case": true, "defn": true, "def": true,
		"assoc": true, "dissoc": true, "get": true, "get-in": true, "update": true, "update-in": true,
		"first": true, "rest": true, "last": true, "count": true, "empty?": true, "seq": true,
		"+": true, "-": true, "*": true, "/": true, "=": true, "<": true, ">": true, "<=": true, ">=": true, "not=": true,
	}
	return coreFunctions[symbol]
}

func (d *DuplicatedCodeAnalyzer) isImportantKeyword(keyword string) bool {
	importantKeywords := map[string]bool{
		"require": true, "import": true, "refer": true, "as": true, "exclude": true, "only": true,
		"keys": true, "vals": true, "strs": true, "syms": true,
	}
	return importantKeywords[keyword]
}

func (d *DuplicatedCodeAnalyzer) isNumericLiteral(value string) bool {
	return numericLiteralRegex.MatchString(value)
}

func (d *DuplicatedCodeAnalyzer) calculateLines(node *reader.RichNode) int {
	if node.Location != nil && node.Location.EndLine > node.Location.StartLine {
		return node.Location.EndLine - node.Location.StartLine + 1
	}
	complexity := d.calculateTokens(node)
	return maxInt(1, complexity/5)
}

func (d *DuplicatedCodeAnalyzer) calculateTokens(node *reader.RichNode) int {
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

func (d *DuplicatedCodeAnalyzer) extractNodeContent(node *reader.RichNode) string {
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
			parts = append(parts, d.extractNodeContent(child))
		}
		parts = append(parts, ")")
	case reader.NodeVector:
		parts = append(parts, "[")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, d.extractNodeContent(child))
		}
		parts = append(parts, "]")
	case reader.NodeMap:
		parts = append(parts, "{")
		for i, child := range node.Children {
			if i > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, d.extractNodeContent(child))
		}
		parts = append(parts, "}")
	default:
		if node.Value != "" {
			parts = append(parts, node.Value)
		}
	}

	return strings.Join(parts, "")
}
