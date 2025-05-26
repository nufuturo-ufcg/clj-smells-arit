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

// GlobalDuplicatedCodeAnalyzer é um analisador global para código duplicado
// Detecta duplicações baseado na estrutura da AST, não apenas texto
type GlobalDuplicatedCodeAnalyzer struct {
	mu          sync.Mutex
	codeBlocks  map[string][]CodeBlockInfo
	minLines    int
	minTokens   int
	initialized bool
}

// CodeBlockInfo contém informações sobre um bloco de código
type CodeBlockInfo struct {
	Hash         string
	Content      string
	NormalizedAST string
	File         string
	Location     *reader.Location
	Lines        int
	Tokens       int
	BlockType    string // "function", "let-block", "when-block", etc.
	Context      string // Nome da função/contexto onde o bloco está
}

var globalAnalyzer *GlobalDuplicatedCodeAnalyzer

// GetGlobalDuplicatedCodeAnalyzer retorna a instância global do analisador
func GetGlobalDuplicatedCodeAnalyzer() *GlobalDuplicatedCodeAnalyzer {
	if globalAnalyzer == nil {
		globalAnalyzer = &GlobalDuplicatedCodeAnalyzer{
			codeBlocks: make(map[string][]CodeBlockInfo),
			minLines:   3,  // Mínimo de 3 linhas
			minTokens:  15, // Mínimo de 15 tokens
		}
	}
	return globalAnalyzer
}

// AnalyzeTree analisa uma árvore de parsing procurando por código duplicado
func (g *GlobalDuplicatedCodeAnalyzer) AnalyzeTree(tree *parse.Tree, richNodes []*reader.RichNode, filepath string) []Finding {
	g.mu.Lock()
	defer g.mu.Unlock()

	var findings []Finding
	
	// Extrai blocos de código de diferentes tipos
	blocks := g.extractAllCodeBlocks(richNodes, filepath)
	
	// Adiciona blocos ao registro global
	for _, block := range blocks {
		g.codeBlocks[block.Hash] = append(g.codeBlocks[block.Hash], block)
		
		// Se encontrou duplicação, cria finding
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

// extractAllCodeBlocks extrai blocos de código de diferentes tipos
func (g *GlobalDuplicatedCodeAnalyzer) extractAllCodeBlocks(nodes []*reader.RichNode, filepath string) []CodeBlockInfo {
	var blocks []CodeBlockInfo
	
	for _, node := range nodes {
		g.extractBlocksFromNode(node, filepath, "", &blocks)
	}
	return blocks
}

// extractBlocksFromNode extrai blocos recursivamente de diferentes tipos de nós
func (g *GlobalDuplicatedCodeAnalyzer) extractBlocksFromNode(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if node == nil {
		return
	}
	
	// Detecta diferentes tipos de blocos de código
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
	
	// Processa recursivamente os filhos
	newContext := context
	if g.isFunctionDefinition(node) && len(node.Children) > 1 {
		newContext = node.Children[1].Value
	}
	
	for _, child := range node.Children {
		g.extractBlocksFromNode(child, filepath, newContext, blocks)
	}
}

// isFunctionDefinition verifica se é uma definição de função
func (g *GlobalDuplicatedCodeAnalyzer) isFunctionDefinition(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "defn" || 
		 node.Children[0].Value == "defn-" ||
		 node.Children[0].Value == "defmacro" ||
		 node.Children[0].Value == "defmethod")
}

// isLetBlock verifica se é um bloco let
func (g *GlobalDuplicatedCodeAnalyzer) isLetBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "let" ||
		 node.Children[0].Value == "when-let" ||
		 node.Children[0].Value == "if-let" ||
		 node.Children[0].Value == "binding")
}

// isConditionalBlock verifica se é um bloco condicional
func (g *GlobalDuplicatedCodeAnalyzer) isConditionalBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "if" ||
		 node.Children[0].Value == "when" ||
		 node.Children[0].Value == "cond" ||
		 node.Children[0].Value == "case" ||
		 node.Children[0].Value == "condp")
}

// isLoopBlock verifica se é um bloco de loop
func (g *GlobalDuplicatedCodeAnalyzer) isLoopBlock(node *reader.RichNode) bool {
	return node.Type == reader.NodeList && len(node.Children) > 0 &&
		node.Children[0].Type == reader.NodeSymbol &&
		(node.Children[0].Value == "loop" ||
		 node.Children[0].Value == "doseq" ||
		 node.Children[0].Value == "dotimes" ||
		 node.Children[0].Value == "for")
}

// isSignificantBlock verifica se é um bloco significativo para análise
func (g *GlobalDuplicatedCodeAnalyzer) isSignificantBlock(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}
	
	// Calcula complexidade do bloco
	complexity := g.calculateComplexity(node)
	return complexity >= g.minTokens
}

// calculateComplexity calcula a complexidade de um nó baseado em sua estrutura
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

// extractFunctionBlock extrai bloco de função
func (g *GlobalDuplicatedCodeAnalyzer) extractFunctionBlock(node *reader.RichNode, filepath string, blocks *[]CodeBlockInfo) {
	g.extractTypedBlock(node, filepath, "function", "", blocks)
}

// extractLetBlock extrai bloco let
func (g *GlobalDuplicatedCodeAnalyzer) extractLetBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if len(node.Children) >= 3 { // let tem pelo menos [bindings] e body
		g.extractTypedBlock(node, filepath, "let-block", context, blocks)
	}
}

// extractConditionalBlock extrai bloco condicional
func (g *GlobalDuplicatedCodeAnalyzer) extractConditionalBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if len(node.Children) >= 2 {
		g.extractTypedBlock(node, filepath, "conditional-block", context, blocks)
	}
}

// extractLoopBlock extrai bloco de loop
func (g *GlobalDuplicatedCodeAnalyzer) extractLoopBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	if len(node.Children) >= 2 {
		g.extractTypedBlock(node, filepath, "loop-block", context, blocks)
	}
}

// extractGenericBlock extrai bloco genérico
func (g *GlobalDuplicatedCodeAnalyzer) extractGenericBlock(node *reader.RichNode, filepath, context string, blocks *[]CodeBlockInfo) {
	g.extractTypedBlock(node, filepath, "generic-block", context, blocks)
}

// extractTypedBlock extrai um bloco tipado
func (g *GlobalDuplicatedCodeAnalyzer) extractTypedBlock(node *reader.RichNode, filepath, blockType, context string, blocks *[]CodeBlockInfo) {
	content := g.extractNodeContent(node)
	normalizedAST := g.normalizeAST(node)
	
	lines := g.calculateLines(node)
	tokens := g.calculateTokens(node)
	
	if lines >= g.minLines && tokens >= g.minTokens {
		hash := fmt.Sprintf("%x", md5.Sum([]byte(normalizedAST)))
		
		block := CodeBlockInfo{
			Hash:         hash,
			Content:      content,
			NormalizedAST: normalizedAST,
			File:         filepath,
			Location:     node.Location,
			Lines:        lines,
			Tokens:       tokens,
			BlockType:    blockType,
			Context:      context,
		}
		
		*blocks = append(*blocks, block)
	}
}

// normalizeAST normaliza a AST para comparação estrutural
func (g *GlobalDuplicatedCodeAnalyzer) normalizeAST(node *reader.RichNode) string {
	if node == nil {
		return ""
	}
	
	var parts []string
	
	switch node.Type {
	case reader.NodeSymbol:
		// Normaliza símbolos baseado em padrões
		normalized := g.normalizeSymbol(node.Value)
		parts = append(parts, normalized)
		
	case reader.NodeString:
		parts = append(parts, "STRING_LITERAL")
		
	case reader.NodeKeyword:
		// Mantém keywords importantes, normaliza outras
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
			// Normaliza valores literais
			if g.isNumericLiteral(node.Value) {
				parts = append(parts, "NUMBER")
			} else {
				parts = append(parts, g.normalizeSymbol(node.Value))
			}
		}
	}
	
	return strings.Join(parts, "")
}

// normalizeSymbol normaliza símbolos baseado em padrões semânticos
func (g *GlobalDuplicatedCodeAnalyzer) normalizeSymbol(symbol string) string {
	// Funções de core do Clojure - mantém
	coreFunction := []string{
		"map", "filter", "reduce", "apply", "partial", "comp",
		"let", "when", "if", "cond", "case", "defn", "def",
		"assoc", "dissoc", "get", "get-in", "update", "update-in",
		"first", "rest", "last", "count", "empty?", "seq",
		"+", "-", "*", "/", "=", "<", ">", "<=", ">=", "not=",
	}
	
	for _, core := range coreFunction {
		if symbol == core {
			return symbol
		}
	}
	
	// Padrões de nomeação - normaliza para tipos genéricos
	patterns := map[string]string{
		// Variáveis de dados
		`^(data|info|result|response|request|payload)$`: "DATA_VAR",
		`^(item|element|entry|record)s?$`:               "ITEM_VAR",
		`^(user|customer|person|entity|account)s?$`:     "ENTITY_VAR",
		`^(id|key|index|idx)$`:                          "ID_VAR",
		`^(name|title|label)$`:                          "NAME_VAR",
		`^(value|val|amount|total|sum)s?$`:              "VALUE_VAR",
		`^(config|settings|options|params?)$`:          "CONFIG_VAR",
		
		// Funções de processamento
		`^(process|handle|manage|execute)-.*`:     "PROCESS_FUNC",
		`^(get|fetch|retrieve|load)-.*`:           "GET_FUNC",
		`^(set|save|store|persist|update)-.*`:     "SET_FUNC",
		`^(validate|check|verify|ensure)-.*`:      "VALIDATE_FUNC",
		`^(calculate|compute|determine|find)-.*`:  "CALC_FUNC",
		`^(create|make|build|generate)-.*`:        "CREATE_FUNC",
		`^(parse|format|transform|convert)-.*`:    "TRANSFORM_FUNC",
	}
	
	for pattern, replacement := range patterns {
		if matched, _ := regexp.MatchString(pattern, symbol); matched {
			return replacement
		}
	}
	
	// Se não matched nenhum padrão, mantém como variável genérica
	return "VAR"
}

// isImportantKeyword verifica se é uma keyword importante para manter
func (g *GlobalDuplicatedCodeAnalyzer) isImportantKeyword(keyword string) bool {
	important := []string{
		"require", "import", "refer", "as", "exclude", "only",
		"keys", "vals", "strs", "syms",
	}
	
	for _, imp := range important {
		if keyword == imp {
			return true
		}
	}
	return false
}

// isNumericLiteral verifica se é um literal numérico
func (g *GlobalDuplicatedCodeAnalyzer) isNumericLiteral(value string) bool {
	matched, _ := regexp.MatchString(`^-?\d+(\.\d+)?$`, value)
	return matched
}

// calculateLines calcula o número de linhas baseado na localização
func (g *GlobalDuplicatedCodeAnalyzer) calculateLines(node *reader.RichNode) int {
	if node.Location != nil && node.Location.EndLine > node.Location.StartLine {
		return node.Location.EndLine - node.Location.StartLine + 1
	}
	
	// Fallback: estima baseado na complexidade
	complexity := g.calculateComplexity(node)
	return max(1, complexity/5) // Aproximadamente 5 tokens por linha
}

// calculateTokens calcula o número de tokens na AST
func (g *GlobalDuplicatedCodeAnalyzer) calculateTokens(node *reader.RichNode) int {
	return g.calculateComplexity(node)
}

// extractNodeContent extrai o conteúdo textual de um nó
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

// createDuplicationMessage cria mensagem de duplicação
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

// max retorna o maior entre dois inteiros
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Reset limpa o estado do analisador global
func (g *GlobalDuplicatedCodeAnalyzer) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.codeBlocks = make(map[string][]CodeBlockInfo)
}

// GetStatistics retorna estatísticas do analisador
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
		"total_blocks":       totalBlocks,
		"unique_hashes":      len(g.codeBlocks),
		"duplicated_hashes":  duplicatedHashes,
		"duplicated_blocks":  duplicatedBlocks,
		"blocks_by_type":     blocksByType,
		"duplication_ratio":  float64(duplicatedBlocks) / float64(totalBlocks),
	}
}

// SetThresholds permite configurar os thresholds mínimos
func (g *GlobalDuplicatedCodeAnalyzer) SetThresholds(minLines, minTokens int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.minLines = minLines
	g.minTokens = minTokens
}

// DuplicatedCodeGlobalRule implementa a regra de código duplicado global
type DuplicatedCodeGlobalRule struct {
	Rule
}

func (r *DuplicatedCodeGlobalRule) Meta() Rule {
	return r.Rule
}

func (r *DuplicatedCodeGlobalRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Esta regra usa análise global, então não processa nós individuais
	// A análise é feita pelo GlobalDuplicatedCodeAnalyzer
	return nil
}

func init() {
	defaultRule := &DuplicatedCodeGlobalRule{
		Rule: Rule{
			ID:          "duplicated-code-global",
			Name:        "Duplicated Code (Global Analysis)",
			Description: "Detects duplicated code blocks across the entire codebase using AST-based structural analysis. " +
				"This rule identifies similar code patterns in functions, let blocks, conditionals, loops, and other significant code structures, " +
				"normalizing variable names and literals to focus on structural similarity rather than exact text matches.",
			Severity: SeverityWarning,
		},
	}

	RegisterRule(defaultRule)
} 