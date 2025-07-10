package rules

import (
	"crypto/md5"
	"fmt"
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
	similarCodeBlocks    map[string][]CodeBlockInfo
	processedNodes       map[*reader.RichNode]string
	nodeCounter          int64
	totalBlocksProcessed int64

	enableExact      bool
	enableSimilar    bool
	exactMinLines    int
	exactMinTokens   int
	similarMinLines  int
	similarMinTokens int
	maxCacheSize     int
	maxBlocksPerFile int
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
	DetectionType string
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
			exactCodeBlocks:   make(map[string][]CodeBlockInfo),
			similarCodeBlocks: make(map[string][]CodeBlockInfo),
			processedNodes:    make(map[*reader.RichNode]string),

			enableExact:      true,
			enableSimilar:    true,
			exactMinLines:    8,
			exactMinTokens:   30,
			similarMinLines:  5,
			similarMinTokens: 20,
			maxCacheSize:     10000,
			maxBlocksPerFile: 1000,
		}
	})
	return unifiedAnalyzer
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

		var exactFound bool
		if d.enableExact && d.meetsExactThresholds(block) {
			exactFindings := d.processExactDuplicate(block, filepath)
			findings = append(findings, exactFindings...)

			exactFound = len(d.exactCodeBlocks[block.Hash]) > 1
		}

		if d.enableSimilar && d.meetsSimilarThresholds(block) {

			if !d.enableExact || !exactFound {
				findings = append(findings, d.processSimilarDuplicate(block, filepath)...)
			}

		}
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
		d.extractTypedBlock(node, filepath, "function", context, blocks, blockCount)
	case d.isLetBlock(node):
		d.extractTypedBlock(node, filepath, "let-block", context, blocks, blockCount)
	case d.isConditionalBlock(node):
		d.extractTypedBlock(node, filepath, "conditional-block", context, blocks, blockCount)
	case d.isLoopBlock(node):
		d.extractTypedBlock(node, filepath, "loop-block", context, blocks, blockCount)
	case d.isSignificantBlock(node):
		d.extractTypedBlock(node, filepath, "generic-block", context, blocks, blockCount)
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

func (d *DuplicatedCodeAnalyzer) extractTypedBlock(node *reader.RichNode, filepath, blockType, context string, blocks *[]CodeBlockInfo, blockCount *int) {
	d.nodeCounter++
	nodeID := d.nodeCounter

	content := d.extractNodeContent(node)
	lines := d.calculateLines(node)
	tokens := d.calculateTokens(node)

	exactMeetsThreshold := d.enableExact && lines >= d.exactMinLines && tokens >= d.exactMinTokens
	similarMeetsThreshold := d.enableSimilar && lines >= d.similarMinLines && tokens >= d.similarMinTokens

	if !exactMeetsThreshold && !similarMeetsThreshold {
		return
	}

	var exactNormalized, similarNormalized string
	var exactHash, similarHash string

	if exactMeetsThreshold {
		exactNormalized = d.normalizeASTWithStrategy(node, true)
		exactHash = fmt.Sprintf("%x", md5.Sum([]byte("exact:"+exactNormalized)))

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
			DetectionType: "exact",
			NodeID:        nodeID,
		}
		*blocks = append(*blocks, block)
	}

	if similarMeetsThreshold {
		similarNormalized = d.normalizeASTWithStrategy(node, false)
		similarHash = fmt.Sprintf("%x", md5.Sum([]byte("similar:"+similarNormalized)))

		if !exactMeetsThreshold || similarHash != exactHash {
			block := CodeBlockInfo{
				Hash:          similarHash,
				Content:       content,
				NormalizedAST: similarNormalized,
				File:          filepath,
				Location:      node.Location,
				Lines:         lines,
				Tokens:        tokens,
				BlockType:     blockType,
				Context:       context,
				DetectionType: "similar",
				NodeID:        nodeID,
			}
			*blocks = append(*blocks, block)
		}
	}

	*blockCount++
	d.totalBlocksProcessed++
}

func (d *DuplicatedCodeAnalyzer) normalizeASTWithStrategy(node *reader.RichNode, exact bool) string {
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
		case reader.NodeList, reader.NodeVector, reader.NodeMap, reader.NodeSet:
			builder.WriteString("(")
			for _, child := range n.Children {
				visit(child)
				builder.WriteString(" ")
			}
			builder.WriteString(")")
		case reader.NodeSymbol:
			if exact {
				builder.WriteString(d.normalizeSymbolExact(n.Value))
			} else {
				builder.WriteString(d.normalizeSymbolSimilar(n.Value))
			}
		case reader.NodeKeyword:
			if exact && d.isImportantKeyword(n.Value) {
				builder.WriteString(n.Value)
			} else {
				builder.WriteString(":KEYWORD")
			}
		case reader.NodeString:
			builder.WriteString("\"STRING\"")
		case reader.NodeNumber:
			builder.WriteString("NUMBER")
		case reader.NodeBool:
			if exact {
				builder.WriteString(n.Value)
			} else {
				builder.WriteString("BOOL")
			}
		default:

		}
	}

	visit(node)
	return builder.String()
}

func (d *DuplicatedCodeAnalyzer) normalizeSymbolExact(symbol string) string {
	if d.isCoreFunctionSymbol(symbol) {
		return symbol
	}
	if d.isNumericLiteral(symbol) {
		return "NUMBER"
	}

	for pattern, replacement := range symbolReplacements {
		if symbolPatterns[pattern].MatchString(symbol) {
			return replacement
		}
	}
	return "VAR"
}

func (d *DuplicatedCodeAnalyzer) normalizeSymbolSimilar(symbol string) string {
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

func (d *DuplicatedCodeAnalyzer) meetsExactThresholds(block CodeBlockInfo) bool {
	return block.Lines >= d.exactMinLines && block.Tokens >= d.exactMinTokens
}

func (d *DuplicatedCodeAnalyzer) meetsSimilarThresholds(block CodeBlockInfo) bool {
	return block.Lines >= d.similarMinLines && block.Tokens >= d.similarMinTokens
}

func (d *DuplicatedCodeAnalyzer) processExactDuplicate(block CodeBlockInfo, filepath string) []Finding {
	if block.DetectionType != "exact" {
		return nil
	}

	var findings []Finding
	d.exactCodeBlocks[block.Hash] = append(d.exactCodeBlocks[block.Hash], block)

	if len(d.exactCodeBlocks[block.Hash]) > 1 {
		finding := Finding{
			RuleID:   "duplicated-code-exact",
			Message:  d.createDuplicationMessage(block, len(d.exactCodeBlocks[block.Hash]), "exact"),
			Filepath: filepath,
			Location: block.Location,
			Severity: SeverityWarning,
		}
		findings = append(findings, finding)
	}

	return findings
}

func (d *DuplicatedCodeAnalyzer) processSimilarDuplicate(block CodeBlockInfo, filepath string) []Finding {
	if block.DetectionType != "similar" {
		return nil
	}

	var findings []Finding
	d.similarCodeBlocks[block.Hash] = append(d.similarCodeBlocks[block.Hash], block)

	if len(d.similarCodeBlocks[block.Hash]) > 1 {
		finding := Finding{
			RuleID:   "duplicated-code-similar",
			Message:  d.createDuplicationMessage(block, len(d.similarCodeBlocks[block.Hash]), "similar"),
			Filepath: filepath,
			Location: block.Location,
			Severity: SeverityWarning,
		}
		findings = append(findings, finding)
	}

	return findings
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

func (d *DuplicatedCodeAnalyzer) createDuplicationMessage(block CodeBlockInfo, count int, detectionType string) string {
	var otherFiles []string
	var blocks []CodeBlockInfo

	if detectionType == "exact" {
		blocks = d.exactCodeBlocks[block.Hash]
	} else {
		blocks = d.similarCodeBlocks[block.Hash]
	}

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

	detectionTypeDesc := "duplicated"
	if detectionType == "exact" {
		detectionTypeDesc = "exactly duplicated"
	} else {
		detectionTypeDesc = "similar"
	}

	titleCase := func(s string) string {
		if len(s) == 0 {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}

	message := fmt.Sprintf("%s %s detected in %q (%d occurrences, %d lines, %d tokens)",
		titleCase(detectionTypeDesc), desc, context, count, block.Lines, block.Tokens)

	if len(otherFiles) > 0 {
		message += fmt.Sprintf(". Also found in: %s", strings.Join(otherFiles, ", "))
	}

	return message
}

func (d *DuplicatedCodeAnalyzer) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.exactCodeBlocks = make(map[string][]CodeBlockInfo)
	d.similarCodeBlocks = make(map[string][]CodeBlockInfo)
	d.processedNodes = make(map[*reader.RichNode]string)
	d.nodeCounter = 0
	d.totalBlocksProcessed = 0
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

	totalSimilarBlocks := 0
	duplicatedSimilarHashes := 0
	for _, blocks := range d.similarCodeBlocks {
		totalSimilarBlocks += len(blocks)
		if len(blocks) > 1 {
			duplicatedSimilarHashes++
		}
	}

	return map[string]interface{}{
		"total_exact_blocks":        totalExactBlocks,
		"total_similar_blocks":      totalSimilarBlocks,
		"duplicated_exact_hashes":   duplicatedExactHashes,
		"duplicated_similar_hashes": duplicatedSimilarHashes,
		"total_blocks_processed":    d.totalBlocksProcessed,
		"enable_exact":              d.enableExact,
		"enable_similar":            d.enableSimilar,
		"exact_min_lines":           d.exactMinLines,
		"exact_min_tokens":          d.exactMinTokens,
		"similar_min_lines":         d.similarMinLines,
		"similar_min_tokens":        d.similarMinTokens,
		"max_cache_size":            d.maxCacheSize,
		"max_blocks_per_file":       d.maxBlocksPerFile,
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type DuplicatedCodeRule struct {
	Rule
}

func (r *DuplicatedCodeRule) Meta() Rule {
	return r.Rule
}

func (r *DuplicatedCodeRule) Check(_ *reader.RichNode, _ map[string]interface{}, _ string) *Finding {

	return nil
}
