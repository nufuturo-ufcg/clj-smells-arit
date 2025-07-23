package rules

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/thlaurentino/arit/internal/reader"
)

type DuplicatedCodeRule struct {
	Rule

	globalBlocks map[string][]CodeBlockInfo
	mu           sync.Mutex

	exactMinLines            int
	exactMinTokens           int
	maxBlocksPerFile         int
	semanticMapNormalization bool
	detectFunctions          bool
	detectCodeBlocks         bool
}

type CodeBlockInfo struct {
	Hash          string
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
	numericLiteralRegex *regexp.Regexp
	regexInitOnce       sync.Once
)

func initRegex() {
	regexInitOnce.Do(func() {
		numericLiteralRegex = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
	})
}

func (r *DuplicatedCodeRule) Meta() Rule {
	return r.Rule
}

func (r *DuplicatedCodeRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	initRegex()

	if !r.shouldAnalyzeNode(node) {
		return nil
	}

	block := r.extractCodeBlock(node, filepath)
	if block == nil {
		return nil
	}

	return r.checkForDuplication(*block, filepath)
}

func (r *DuplicatedCodeRule) shouldAnalyzeNode(node *reader.RichNode) bool {
	if node == nil {
		return false
	}

	if r.detectFunctions && r.isFunctionDefinition(node) {
		return true
	}

	if r.detectCodeBlocks {
		return r.isLetBlock(node) || r.isConditionalBlock(node) || r.isLoopBlock(node) || r.isSignificantBlock(node)
	}

	return false
}

func (r *DuplicatedCodeRule) extractCodeBlock(node *reader.RichNode, filepath string) *CodeBlockInfo {
	lines := r.calculateLines(node)
	tokens := r.calculateTokens(node)

	if lines < r.exactMinLines || tokens < r.exactMinTokens {
		return nil
	}

	normalizedAST := r.normalizeAST(node)
	hash := r.fastHash("exact:" + normalizedAST)

	blockType := r.getBlockType(node)
	context := r.getContext(node)

	return &CodeBlockInfo{
		Hash:          hash,
		NormalizedAST: normalizedAST,
		File:          filepath,
		Location:      node.Location,
		Lines:         lines,
		Tokens:        tokens,
		BlockType:     blockType,
		Context:       context,
		NodeID:        0,
	}
}

func (r *DuplicatedCodeRule) checkForDuplication(block CodeBlockInfo, filepath string) *Finding {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.globalBlocks == nil {
		r.globalBlocks = make(map[string][]CodeBlockInfo)
	}

	r.globalBlocks[block.Hash] = append(r.globalBlocks[block.Hash], block)

	blocks := r.globalBlocks[block.Hash]
	if len(blocks) <= 1 {
		return nil
	}

	message := r.createDuplicationMessage(block, len(blocks))

	return &Finding{
		RuleID:   r.Rule.ID,
		Message:  message,
		Filepath: filepath,
		Location: block.Location,
		Severity: r.Rule.Severity,
	}
}

func (r *DuplicatedCodeRule) getBlockType(node *reader.RichNode) string {
	if r.isFunctionDefinition(node) {
		return "function"
	}
	if r.isLetBlock(node) {
		return "let-block"
	}
	if r.isConditionalBlock(node) {
		return "conditional-block"
	}
	if r.isLoopBlock(node) {
		return "loop-block"
	}
	return "generic-block"
}

func (r *DuplicatedCodeRule) getContext(node *reader.RichNode) string {
	if r.isFunctionDefinition(node) && len(node.Children) > 1 && node.Children[1] != nil {
		return node.Children[1].Value
	}
	return "unknown"
}

func (r *DuplicatedCodeRule) normalizeAST(node *reader.RichNode) string {
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
			if r.semanticMapNormalization {
				builder.WriteString("{")
				pairs := r.extractAndSortMapPairs(n.Children)
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
			builder.WriteString(r.normalizeSymbol(n.Value))
		case reader.NodeKeyword:
			if r.isImportantKeyword(n.Value) {
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

type mapPair struct {
	key       *reader.RichNode
	value     *reader.RichNode
	keyString string
}

func (r *DuplicatedCodeRule) extractAndSortMapPairs(children []*reader.RichNode) []mapPair {
	var pairs []mapPair

	for i := 0; i < len(children); i += 2 {
		if i+1 < len(children) {
			key := children[i]
			value := children[i+1]

			keyString := key.Value

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

func (r *DuplicatedCodeRule) normalizeSymbol(symbol string) string {
	if r.isCoreFunctionSymbol(symbol) {
		return symbol
	}
	if r.isNumericLiteral(symbol) {
		return "NUMBER"
	}
	return "VAR"
}

func (r *DuplicatedCodeRule) isFunctionDefinition(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" ||
			node.Children[0].Value == "defn-" ||
			node.Children[0].Value == "defmacro" ||
			node.Children[0].Value == "defmethod")
}

func (r *DuplicatedCodeRule) isLetBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "let" ||
			node.Children[0].Value == "when-let" ||
			node.Children[0].Value == "if-let" ||
			node.Children[0].Value == "binding")
}

func (r *DuplicatedCodeRule) isConditionalBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "if" ||
			node.Children[0].Value == "when" ||
			node.Children[0].Value == "cond" ||
			node.Children[0].Value == "case" ||
			node.Children[0].Value == "condp")
}

func (r *DuplicatedCodeRule) isLoopBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "loop" ||
			node.Children[0].Value == "doseq" ||
			node.Children[0].Value == "dotimes" ||
			node.Children[0].Value == "for")
}

func (r *DuplicatedCodeRule) isSignificantBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) >= 4
}

func (r *DuplicatedCodeRule) createDuplicationMessage(block CodeBlockInfo, count int) string {
	var otherFiles []string
	blocks := r.globalBlocks[block.Hash]

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

func (r *DuplicatedCodeRule) fastHash(data string) string {
	h := fnv.New64a()
	_, err := h.Write([]byte(data))
	if err != nil {
		return fmt.Sprintf("%x", len(data))
	}
	return fmt.Sprintf("%x", h.Sum64())
}

func (r *DuplicatedCodeRule) isCoreFunctionSymbol(symbol string) bool {
	coreFunctions := map[string]bool{
		"map": true, "filter": true, "reduce": true, "apply": true, "partial": true, "comp": true,
		"let": true, "when": true, "if": true, "cond": true, "case": true, "defn": true, "def": true,
		"assoc": true, "dissoc": true, "get": true, "get-in": true, "update": true, "update-in": true,
		"first": true, "rest": true, "last": true, "count": true, "empty?": true, "seq": true,
		"+": true, "-": true, "*": true, "/": true, "=": true, "<": true, ">": true, "<=": true, ">=": true, "not=": true,
	}
	return coreFunctions[symbol]
}

func (r *DuplicatedCodeRule) isImportantKeyword(keyword string) bool {
	importantKeywords := map[string]bool{
		"require": true, "import": true, "refer": true, "as": true, "exclude": true, "only": true,
		"keys": true, "vals": true, "strs": true, "syms": true,
	}
	return importantKeywords[keyword]
}

func (r *DuplicatedCodeRule) isNumericLiteral(value string) bool {
	return numericLiteralRegex.MatchString(value)
}

func (r *DuplicatedCodeRule) calculateLines(node *reader.RichNode) int {
	if node.Location != nil && node.Location.EndLine > node.Location.StartLine {
		return node.Location.EndLine - node.Location.StartLine + 1
	}
	complexity := r.calculateTokens(node)
	return maxInt(1, complexity/5)
}

func (r *DuplicatedCodeRule) calculateTokens(node *reader.RichNode) int {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *DuplicatedCodeRule) ResetGlobalState() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.globalBlocks = make(map[string][]CodeBlockInfo)
}

func (r *DuplicatedCodeRule) GetStatistics() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	totalBlocks := 0
	duplicatedHashes := 0
	for _, blocks := range r.globalBlocks {
		totalBlocks += len(blocks)
		if len(blocks) > 1 {
			duplicatedHashes++
		}
	}

	return map[string]interface{}{
		"total_blocks":      totalBlocks,
		"duplicated_hashes": duplicatedHashes,
		"unique_hashes":     len(r.globalBlocks),
	}
}

func init() {
	RegisterRule(&DuplicatedCodeRule{
		Rule: Rule{
			ID:          "duplicated-code",
			Name:        "Duplicated Code Detection",
			Description: "Detects duplicated code blocks that are identical in structure",
			Severity:    SeverityWarning,
		},
		globalBlocks:             make(map[string][]CodeBlockInfo),
		exactMinLines:            1,
		exactMinTokens:           5,
		maxBlocksPerFile:         1000,
		semanticMapNormalization: true,
		detectFunctions:          true,
		detectCodeBlocks:         false,
	})
}
