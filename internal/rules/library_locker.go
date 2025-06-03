// Package rules implementa regras para detectar wrappers desnecessários de bibliotecas externas
// Esta regra específica identifica "Library Locker" - funções que apenas encapsulam chamadas de biblioteca
package rules

import (
	"fmt"
	"strings"

	"github.com/thlaurentino/arit/internal/reader"
)

// LibraryLockerRule detecta funções que apenas fazem wrapper de bibliotecas externas
// Este code smell ocorre quando uma aplicação encapsula uma biblioteca terceirizada
// com suas próprias funções, frequentemente obscurecendo ou complicando o uso da biblioteca
type LibraryLockerRule struct {
	Rule
	ExcludedLibraries []string `json:"excluded_libraries" yaml:"excluded_libraries"` // Bibliotecas a ignorar
	MinParamCount     int      `json:"min_param_count" yaml:"min_param_count"`       // Mínimo de parâmetros para considerar
}

func (r *LibraryLockerRule) Meta() Rule {
	return r.Rule
}

// Check analisa definições de função procurando por library lockers
// Detecta funções que apenas delegam para bibliotecas externas sem agregar valor significativo
func (r *LibraryLockerRule) Check(node *reader.RichNode, context map[string]interface{}, filepath string) *Finding {
	// Verifica se é uma definição de função (defn ou defn-)
	if node.Type != reader.NodeList || len(node.Children) < 3 {
		return nil
	}

	firstChild := node.Children[0]
	if firstChild.Type != reader.NodeSymbol || (firstChild.Value != "defn" && firstChild.Value != "defn-") {
		return nil
	}

	// Extrai nome da função
	funcNameNode := node.Children[1]
	if funcNameNode.Type != reader.NodeSymbol {
		return nil
	}
	funcName := funcNameNode.Value

	// Localiza vetor de parâmetros (pode ter docstring)
	var paramsNode *reader.RichNode
	var bodyStartIndex int

	for i := 2; i < len(node.Children); i++ {
		child := node.Children[i]
		if child.Type == reader.NodeVector {
			paramsNode = child
			bodyStartIndex = i + 1
			break
		}
	}

	if paramsNode == nil || bodyStartIndex >= len(node.Children) {
		return nil
	}

	// Extrai parâmetros
	params := paramsNode.Children

	// Ignora funções com poucos parâmetros (menos propensas a serem library lockers)
	if len(params) < r.MinParamCount {
		return nil
	}

	// Analisa corpo da função procurando por padrão de library locker
	bodyNodes := node.Children[bodyStartIndex:]

	// Procura por único nó significativo no corpo
	var significantBodyNode *reader.RichNode
	for _, bodyNode := range bodyNodes {
		if bodyNode.Type != reader.NodeComment && bodyNode.Type != reader.NodeNewline {
			if significantBodyNode != nil {
				// Múltiplos nós significativos - não é um library locker simples
				return nil
			}
			significantBodyNode = bodyNode
		}
	}

	if significantBodyNode == nil {
		return nil
	}

	// Verifica se o corpo é uma chamada de biblioteca externa
	libraryCall := r.extractLibraryCall(significantBodyNode)
	if libraryCall == nil {
		return nil
	}

	// Verifica se é um library locker válido
	if r.isLibraryLocker(params, libraryCall, funcName) {
		return &Finding{
			RuleID:   r.ID,
			Message:  r.formatMessage(funcName, libraryCall),
			Filepath: filepath,
			Location: node.Location,
			Severity: r.Severity,
		}
	}

	return nil
}

// LibraryCall representa uma chamada para biblioteca externa
type LibraryCall struct {
	Library    string             // Nome da biblioteca (namespace)
	Function   string             // Nome da função
	Arguments  []*reader.RichNode // Argumentos da chamada
	FullSymbol string             // Símbolo completo (lib/func)
}

// extractLibraryCall extrai informações de uma chamada de biblioteca externa
func (r *LibraryLockerRule) extractLibraryCall(node *reader.RichNode) *LibraryCall {
	if node.Type != reader.NodeList || len(node.Children) == 0 {
		return nil
	}

	funcNode := node.Children[0]
	if funcNode.Type != reader.NodeSymbol {
		return nil
	}

	// Verifica se é uma chamada para biblioteca externa (contém namespace)
	funcSymbol := funcNode.Value
	if !strings.Contains(funcSymbol, "/") {
		return nil
	}

	parts := strings.SplitN(funcSymbol, "/", 2)
	if len(parts) != 2 {
		return nil
	}

	library := parts[0]
	function := parts[1]

	// Verifica se está na lista de bibliotecas excluídas
	for _, excluded := range r.ExcludedLibraries {
		if library == excluded {
			return nil
		}
	}

	return &LibraryCall{
		Library:    library,
		Function:   function,
		Arguments:  node.Children[1:],
		FullSymbol: funcSymbol,
	}
}

// isLibraryLocker verifica se a função é um library locker
func (r *LibraryLockerRule) isLibraryLocker(params []*reader.RichNode, call *LibraryCall, funcName string) bool {
	// Verifica correspondência de parâmetros simples
	if r.hasSimpleParameterDelegation(params, call.Arguments) {
		return true
	}

	// Verifica se é um wrapper com configuração adicional
	if r.hasConfiguredParameterDelegation(params, call.Arguments) {
		return true
	}

	// Verifica se é um wrapper que apenas reorganiza parâmetros
	if r.hasReorganizedParameterDelegation(params, call.Arguments) {
		return true
	}

	return false
}

// hasSimpleParameterDelegation verifica delegação direta de parâmetros
func (r *LibraryLockerRule) hasSimpleParameterDelegation(params []*reader.RichNode, args []*reader.RichNode) bool {
	if len(params) != len(args) {
		return false
	}

	for i, param := range params {
		if param.Type != reader.NodeSymbol || args[i].Type != reader.NodeSymbol {
			return false
		}
		if param.Value != args[i].Value {
			return false
		}
	}

	return true
}

// hasConfiguredParameterDelegation verifica delegação com configuração adicional
func (r *LibraryLockerRule) hasConfiguredParameterDelegation(params []*reader.RichNode, args []*reader.RichNode) bool {
	// Permite alguns argumentos constantes no início (configuração)
	if len(args) < len(params) {
		return false
	}

	// Verifica se os últimos argumentos correspondem aos parâmetros
	paramOffset := len(args) - len(params)
	for i, param := range params {
		if param.Type != reader.NodeSymbol {
			return false
		}

		argIndex := paramOffset + i
		arg := args[argIndex]

		if arg.Type != reader.NodeSymbol || param.Value != arg.Value {
			return false
		}
	}

	// Verifica se os argumentos iniciais são constantes (configuração)
	for i := 0; i < paramOffset; i++ {
		arg := args[i]
		// Constantes típicas: keywords, strings, números, maps literais
		if arg.Type != reader.NodeKeyword &&
			arg.Type != reader.NodeString &&
			arg.Type != reader.NodeNumber &&
			arg.Type != reader.NodeMap &&
			arg.Type != reader.NodeBool {
			return false
		}
	}

	return true
}

// hasReorganizedParameterDelegation verifica se apenas reorganiza parâmetros
func (r *LibraryLockerRule) hasReorganizedParameterDelegation(params []*reader.RichNode, args []*reader.RichNode) bool {
	if len(params) != len(args) {
		return false
	}

	// Cria mapa de parâmetros para verificar se todos são usados
	paramMap := make(map[string]bool)
	for _, param := range params {
		if param.Type == reader.NodeSymbol {
			paramMap[param.Value] = false
		}
	}

	// Verifica se todos os argumentos são parâmetros (possivelmente reordenados)
	for _, arg := range args {
		if arg.Type != reader.NodeSymbol {
			return false
		}
		if _, exists := paramMap[arg.Value]; !exists {
			return false
		}
		paramMap[arg.Value] = true
	}

	// Verifica se todos os parâmetros foram usados
	for _, used := range paramMap {
		if !used {
			return false
		}
	}

	return true
}

// formatMessage formata a mensagem do finding
func (r *LibraryLockerRule) formatMessage(funcName string, call *LibraryCall) string {
	return fmt.Sprintf(
		"Function %q appears to be a 'Library Locker' - it merely wraps %q without adding significant value. "+
			"Consider using %q directly or adding meaningful abstraction if the wrapper serves a specific purpose.",
		funcName, call.FullSymbol, call.FullSymbol,
	)
}

// init registra a regra Library Locker com configurações padrão
func init() {
	defaultRule := &LibraryLockerRule{
		Rule: Rule{
			ID:          "library-locker",
			Name:        "Library Locker",
			Description: "Detects functions that unnecessarily wrap third-party library calls without adding meaningful abstraction. This pattern obscures the library's usage and adds unnecessary indirection.",
			Severity:    SeverityInfo,
		},
		ExcludedLibraries: []string{
			"clojure.core",   // Core library é idiomática
			"clojure.string", // String utilities são comuns
			"clojure.set",    // Set operations são básicas
			"clojure.walk",   // Walk é frequentemente wrappado
		},
		MinParamCount: 1, // Pelo menos 1 parâmetro para considerar
	}

	RegisterRule(defaultRule)
}
