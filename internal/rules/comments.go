// Package rules implementa regras de análise para qualidade de comentários em código Clojure
// Esta regra específica detecta problemas comuns em comentários como redundância, obviedade e código comentado
package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// CommentType enumera os diferentes tipos de problemas em comentários
// Usado para categorizar e reportar issues específicos de qualidade
type CommentType int

const (
	CommentEmpty         CommentType = iota // Comentários vazios ou apenas com ";"
	CommentTODO                             // Comentários TODO que deveriam ser issues
	CommentFIXME                            // Comentários FIXME que indicam problemas
	CommentRedundant                        // Comentários que apenas repetem o código
	CommentObvious                          // Comentários que afirmam o óbvio
	CommentDeodorant                        // Comentários que tentam mascarar código ruim
	CommentCommentedCode                    // Código comentado que deveria ser removido
	CommentDocstring                        // Funções que deveriam ter docstring
)

// CommentsRule implementa análise de qualidade de comentários
// Configurável para diferentes tipos de verificação e limites de tamanho
type CommentsRule struct {
	Rule
	ReportTODO          bool `json:"report_todo" yaml:"report_todo"`                     // Reporta comentários TODO
	ReportFIXME         bool `json:"report_fixme" yaml:"report_fixme"`                   // Reporta comentários FIXME
	ReportEmpty         bool `json:"report_empty" yaml:"report_empty"`                   // Reporta comentários vazios
	ReportRedundant     bool `json:"report_redundant" yaml:"report_redundant"`           // Reporta comentários redundantes
	ReportObvious       bool `json:"report_obvious" yaml:"report_obvious"`               // Reporta comentários óbvios
	ReportDeodorant     bool `json:"report_deodorant" yaml:"report_deodorant"`           // Reporta comentários "desodorante"
	ReportCommentedCode bool `json:"report_commented_code" yaml:"report_commented_code"` // Reporta código comentado
	ReportDocstring     bool `json:"report_docstring" yaml:"report_docstring"`           // Reporta funções sem docstring
	MinCommentLength    int  `json:"min_comment_length" yaml:"min_comment_length"`       // Tamanho mínimo de comentário útil
	MaxCommentLength    int  `json:"max_comment_length" yaml:"max_comment_length"`       // Tamanho máximo antes de sugerir docstring
}

func (r *CommentsRule) Meta() Rule {
	return r.Rule
}

// isClojureCode detecta se o comentário contém código Clojure comentado
// Usa padrões regex para identificar estruturas sintáticas típicas do Clojure
func isClojureCode(comment string) bool {
	// Remove prefixos de comentário para análise do conteúdo
	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))

	// Padrões que indicam código Clojure comentado
	// Inclui estruturas de dados, definições e formas especiais
	codePatterns := []string{
		`^\s*\(`,           // Começa com parênteses (listas)
		`^\s*\[`,           // Começa with colchetes (vetores)
		`^\s*\{`,           // Começa com chaves (mapas)
		`^\s*def[a-z-]*\s`, // Definições (defn, def, defmacro, etc.)
		`^\s*let\s*\[`,     // Let bindings
		`^\s*if\s*\(`,      // Condicionais if
		`^\s*when\s*\(`,    // Condicionais when
		`^\s*cond\s*$`,     // Expressões cond
		`^\s*case\s*\(`,    // Expressões case
		`^\s*loop\s*\[`,    // Loops com bindings
		`^\s*recur\s*\(`,   // Chamadas recur
		`^\s*->\s*\(`,      // Threading macros ->
		`^\s*->>\s*\(`,     // Threading macros ->>
		`^\s*require\s*\[`, // Declarações require
		`^\s*import\s*\[`,  // Declarações import
		`^\s*ns\s+[a-z]`,   // Declarações de namespace
	}

	// Verifica cada padrão contra o comentário limpo
	for _, pattern := range codePatterns {
		if matched, _ := regexp.MatchString(pattern, cleaned); matched {
			return true
		}
	}

	// Verifica se contém múltiplas expressões s-expression
	// Indicativo de bloco de código comentado
	parenCount := strings.Count(cleaned, "(") + strings.Count(cleaned, "[") + strings.Count(cleaned, "{")
	if parenCount >= 2 {
		return true
	}

	return false
}

// isRedundantComment detecta comentários redundantes específicos para Clojure
// Identifica comentários que apenas repetem o que o código já expressa claramente
func isRedundantComment(comment string, nextNode *reader.RichNode) bool {
	// Limpa o comentário removendo prefixos
	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))
	commentLower := strings.ToLower(cleaned)

	// Padrões redundantes gerais que apenas descrevem ações
	redundantPatterns := []string{
		"function to",   // "function to calculate..."
		"method to",     // "method to process..."
		"this function", // "this function does..."
		"this method",   // "this method handles..."
		"calls",         // "calls the API"
		"returns",       // "returns the result"
		"defines",       // "defines a variable"
		"creates",       // "creates a new object"
		"sets",          // "sets the value"
		"gets",          // "gets the value"
		"increments",    // "increments the counter"
		"decrements",    // "decrements the counter"
	}

	for _, pattern := range redundantPatterns {
		if strings.Contains(commentLower, pattern) {
			return true
		}
	}

	// Verifica redundância específica do Clojure baseada no próximo nó
	// Analisa se o comentário apenas repete o nome da função/forma especial
	if nextNode != nil && nextNode.Type == reader.NodeList {
		if len(nextNode.Children) > 0 {
			firstChild := nextNode.Children[0]
			if firstChild.Type == reader.NodeSymbol {
				symbol := strings.ToLower(firstChild.Value)

				// Comentários redundantes para definições
				if strings.HasPrefix(symbol, "def") {
					if strings.Contains(commentLower, "define") ||
						strings.Contains(commentLower, "definition") ||
						strings.Contains(commentLower, symbol) {
						return true
					}
				}

				// Comentários redundantes para let bindings
				if symbol == "let" && strings.Contains(commentLower, "let") {
					return true
				}

				// Comentários redundantes para condicionais
				if (symbol == "if" || symbol == "when") &&
					(strings.Contains(commentLower, "if") || strings.Contains(commentLower, "when")) {
					return true
				}
			}
		}
	}

	return false
}

// isObviousComment detecta comentários óbvios que não agregam valor
// Identifica comentários que afirmam coisas que já são claras pelo código
func isObviousComment(comment string) bool {
	// Limpa o comentário para análise
	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))
	commentLower := strings.ToLower(cleaned)

	// Padrões de comentários óbvios que não agregam informação
	obviousPatterns := []string{
		"initialize",       // "initialize the variable"
		"start",            // "start the process"
		"end",              // "end the process"
		"validate",         // "validate the input"
		"check",            // "check the condition"
		"store",            // "store the value"
		"save",             // "save the data"
		"transform",        // "transform the data"
		"convert",          // "convert to string"
		"process",          // "process the request"
		"handle",           // "handle the event"
		"manage",           // "manage the state"
		"update",           // "update the record"
		"delete",           // "delete the item"
		"create",           // "create new instance"
		"add",              // "add to collection"
		"remove",           // "remove from collection"
		"get",              // "get the value"
		"set",              // "set the value"
		"main function",    // "main function"
		"helper function",  // "helper function"
		"utility function", // "utility function"
	}

	// Verifica se o comentário contém algum padrão óbvio
	for _, pattern := range obviousPatterns {
		if strings.Contains(commentLower, pattern) {
			return true
		}
	}

	return false
}

// isDeodorantComment detecta comentários "desodorante" que tentam mascarar código ruim
// Estes comentários indicam que o código precisa de refatoração, não de explicação
func isDeodorantComment(comment string) bool {
	// Limpa o comentário para análise
	cleaned := strings.TrimSpace(strings.TrimPrefix(comment, ";"))
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, ";;"))
	commentLower := strings.ToLower(cleaned)

	// Padrões que indicam comentários "desodorante"
	// Estes comentários sugerem problemas no código que deveriam ser corrigidos
	deodorantPatterns := []string{
		"hack",         // "this is a hack"
		"workaround",   // "workaround for bug"
		"temporary",    // "temporary solution"
		"quick fix",    // "quick fix for now"
		"dirty",        // "dirty implementation"
		"ugly",         // "ugly but works"
		"bad",          // "bad code but..."
		"terrible",     // "terrible solution"
		"awful",        // "awful implementation"
		"sorry",        // "sorry for this code"
		"apologize",    // "I apologize for..."
		"don't ask",    // "don't ask why"
		"magic number", // "magic number here"
		"hardcoded",    // "hardcoded value"
		"hard coded",   // "hard coded solution"
		"kludge",       // "kludge to make it work"
		"bodge",        // "bodge job"
		"duct tape",    // "duct tape solution"
	}

	// Verifica se o comentário contém linguagem que indica código problemático
	for _, pattern := range deodorantPatterns {
		if strings.Contains(commentLower, pattern) {
			return true
		}
	}

	return false
}

// shouldHaveDocstring verifica se uma função deveria ter docstring
// Funções públicas e macros deveriam ter documentação adequada
func shouldHaveDocstring(node *reader.RichNode) bool {
	if node.Type != reader.NodeList || len(node.Children) < 2 {
		return false
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol {
		return false
	}

	// Tipos de definição que deveriam ter docstring
	// Inclui funções, macros, protocolos e tipos
	defTypes := []string{"defn", "defn-", "defmacro", "defprotocol", "defrecord", "deftype"}
	symbol := firstChild.Value

	for _, defType := range defTypes {
		if symbol == defType {
			// Verifica se já tem docstring (string como terceiro elemento)
			if len(node.Children) >= 3 && node.Children[2].Type == reader.NodeString {
				return false // Já tem docstring
			}
			return true // Deveria ter mas não tem
		}
	}

	return false
}

// Check implementa a interface CheckerRule para análise de comentários
// Verifica apenas nós de comentário, delegando a análise específica
func (r *CommentsRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica apenas nós de comentário
	if node.Type == reader.NodeComment {
		return r.checkComment(node, context, filepath)
	}

	return nil
}

// checkComment realiza a análise detalhada de um comentário específico
// Aplica todas as verificações configuradas e retorna findings apropriados
func (r *CommentsRule) checkComment(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	commentText := strings.TrimSpace(node.Value)
	currentLocation := node.Location

	// Verifica comentários vazios (apenas ";" ou ";;")
	if r.ReportEmpty && (commentText == "" || commentText == ";" || commentText == ";;") {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Empty comment found. Consider removing it.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	// Verifica tamanho dos comentários (muito curtos ou muito longos)
	cleanedComment := strings.TrimSpace(strings.TrimPrefix(commentText, ";"))
	cleanedComment = strings.TrimSpace(strings.TrimPrefix(cleanedComment, ";;"))

	// Comentários muito curtos provavelmente não são úteis
	if r.MinCommentLength > 0 && len(cleanedComment) < r.MinCommentLength && len(cleanedComment) > 0 {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Comment is too short (%d chars). Consider providing more context or removing it.", len(cleanedComment)),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	// Comentários muito longos deveriam ser docstrings
	if r.MaxCommentLength > 0 && len(cleanedComment) > r.MaxCommentLength {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("Comment is too long (%d chars). Consider breaking it into multiple lines or using docstrings.", len(cleanedComment)),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	// Verifica comentários TODO/FIXME que deveriam ser issues/tasks
	commentUpper := strings.ToUpper(commentText)
	if r.ReportTODO && strings.Contains(commentUpper, "TODO") {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("TODO comment found: %s. Consider creating an issue/task instead.", commentText),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	if r.ReportFIXME && strings.Contains(commentUpper, "FIXME") {
		return &Finding{
			RuleID:   r.ID,
			Message:  fmt.Sprintf("FIXME comment found: %s. Consider creating an issue/task instead.", commentText),
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityWarning,
		}
	}

	// Verifica código comentado que deveria ser removido
	if r.ReportCommentedCode && isClojureCode(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Commented-out code found. Consider removing it or using version control instead.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityWarning,
		}
	}

	// Verifica comentários "desodorante" que indicam código problemático
	if r.ReportDeodorant && isDeodorantComment(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Deodorant comment found. This comment suggests the code needs refactoring rather than explanation.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityWarning,
		}
	}

	// Obtém próximo nó para análise de redundância
	// Necessário para verificar se o comentário apenas repete o código seguinte
	var nextNode *reader.RichNode
	if parent := context["parent"]; parent != nil {
		if parentNode, ok := parent.(*reader.RichNode); ok {
			for i, child := range parentNode.Children {
				if child == node && i+1 < len(parentNode.Children) {
					nextNode = parentNode.Children[i+1]
					break
				}
			}
		}
	}

	// Verifica comentários redundantes que apenas repetem o código
	if r.ReportRedundant && isRedundantComment(commentText, nextNode) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Redundant comment that just describes what the code does. Consider explaining 'why' instead of 'what', or remove it.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	// Verifica comentários óbvios que não agregam valor
	if r.ReportObvious && isObviousComment(commentText) {
		return &Finding{
			RuleID:   r.ID,
			Message:  "Comment states the obvious. Consider removing it or providing more meaningful information.",
			Filepath: filepath,
			Location: currentLocation,
			Severity: SeverityInfo,
		}
	}

	return nil
}

// init registra a regra de comentários com configurações padrão
// Configuração balanceada para detectar problemas sem gerar muito ruído
func init() {
	defaultRule := &CommentsRule{
		Rule: Rule{
			ID:          "comments",
			Name:        "Comment Quality Analysis",
			Description: "Analyzes comments for quality issues in Clojure code. Detects redundant, obvious, deodorant comments, commented-out code, and missing docstrings. Promotes self-documenting code through good naming and structure. Comments should explain 'why' not 'what'.",
			Severity:    SeverityInfo,
		},
		ReportTODO:          true,  // Reporta TODOs que deveriam ser issues
		ReportFIXME:         true,  // Reporta FIXMEs que indicam problemas
		ReportEmpty:         true,  // Reporta comentários vazios
		ReportRedundant:     true,  // Reporta comentários redundantes
		ReportObvious:       true,  // Reporta comentários óbvios
		ReportDeodorant:     true,  // Reporta comentários que mascaram código ruim
		ReportCommentedCode: true,  // Reporta código comentado
		ReportDocstring:     false, // Desabilitado por padrão para evitar ruído
		MinCommentLength:    5,     // Comentários muito curtos provavelmente não são úteis
		MaxCommentLength:    120,   // Comentários muito longos deveriam ser docstrings
	}

	RegisterRule(defaultRule)
}
