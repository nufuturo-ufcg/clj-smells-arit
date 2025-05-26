// Package reader implementa o parser e estruturas de dados para análise de código Clojure
// Fornece uma representação rica da AST (Abstract Syntax Tree) com informações de localização e metadados
package reader

import "github.com/cespare/goclj/parse"

// NodeType representa os diferentes tipos de nós na AST do Clojure
type NodeType string

// Constantes para os tipos básicos de nós Clojure
const (
	// Estruturas de dados básicas
	NodeList   NodeType = "List"   // Listas: (1 2 3)
	NodeVector NodeType = "Vector" // Vetores: [1 2 3]
	NodeMap    NodeType = "Map"    // Mapas: {:a 1 :b 2}
	NodeSet    NodeType = "Set"    // Conjuntos: #{1 2 3}

	// Tipos de dados primitivos
	NodeSymbol    NodeType = "Symbol"    // Símbolos: foo, bar
	NodeKeyword   NodeType = "Keyword"   // Palavras-chave: :foo, :bar
	NodeString    NodeType = "String"    // Strings: "hello"
	NodeNumber    NodeType = "Number"    // Números: 42, 3.14
	NodeBool      NodeType = "Bool"      // Booleanos: true, false
	NodeNil       NodeType = "Nil"       // Valor nulo: nil
	NodeCharacter NodeType = "Character" // Caracteres: \a, \newline
	NodeComment   NodeType = "Comment"   // Comentários: ; comentário
	NodeMetadata  NodeType = "Metadata"  // Metadados: ^{:doc "..."}
	NodeTag       NodeType = "Tag"       // Tags: #inst, #uuid
	NodeRegex     NodeType = "Regex"     // Expressões regulares: #"pattern"

	// Macros especiais e reader macros
	NodeQuote            NodeType = "Quote"            // Quote: 'expr
	NodeSyntaxQuote      NodeType = "SyntaxQuote"      // Syntax quote: `expr
	NodeUnquote          NodeType = "Unquote"          // Unquote: ~expr
	NodeUnquoteSplice    NodeType = "UnquoteSplice"    // Unquote splice: ~@expr
	NodeDeref            NodeType = "Deref"            // Deref: @expr
	NodeVarQuote         NodeType = "VarQuote"         // Var quote: #'var
	NodeFnLiteral        NodeType = "FnLiteral"        // Function literal: #(+ % 1)
	NodeReaderCond       NodeType = "ReaderCond"       // Reader conditional: #?(:clj ...)
	NodeReaderCondSplice NodeType = "ReaderCondSplice" // Reader cond splice: #?@(:clj ...)
	NodeReaderDiscard    NodeType = "ReaderDiscard"    // Reader discard: #_expr
	NodeReaderEval       NodeType = "ReaderEval"       // Reader eval: #=expr

	// Tipos especiais para formatação
	NodeNewline NodeType = "Newline" // Quebras de linha
	NodeUnknown NodeType = "Unknown" // Nós não reconhecidos
)

// Location representa a posição de um nó no código fonte
// Inclui linha e coluna de início e fim para rastreamento preciso
type Location struct {
	StartLine   int `json:"start_line"` // Linha de início (1-indexed)
	StartColumn int `json:"start_col"`  // Coluna de início (1-indexed)
	EndLine     int `json:"end_line"`   // Linha de fim (1-indexed)
	EndColumn   int `json:"end_col"`    // Coluna de fim (1-indexed)
}

// RichNode representa um nó enriquecido da AST com informações adicionais
// Estende a funcionalidade básica do parser com metadados úteis para análise
type RichNode struct {
	Type     NodeType    `json:"type"`                // Tipo do nó
	Value    string      `json:"value,omitempty"`     // Valor textual do nó
	Location *Location   `json:"location,omitempty"`  // Localização no código fonte
	Children []*RichNode `json:"children,omitempty"`  // Nós filhos
	Metadata *RichNode   `json:"metadata,omitempty"`  // Metadados associados
	Comments []*RichNode `json:"comments,omitempty"`  // Comentários associados
	TypeHint string      `json:"type_hint,omitempty"` // Dica de tipo (se presente)

	// ResolvedDefinition aponta para a definição resolvida de um símbolo
	// Usado para análise semântica e detecção de referências
	ResolvedDefinition *RichNode `json:"-"`

	// InferredType contém o tipo inferido durante análise estática
	InferredType string `json:"inferred_type,omitempty"`

	// OriginalNode mantém referência ao nó original do parser
	// Útil para operações que precisam acessar a implementação original
	OriginalNode parse.Node `json:"-"`

	// Scope e SymbolRef são usados para análise de escopo e resolução de símbolos
	// Implementação específica pode variar dependendo das necessidades de análise
	Scope     interface{} // Escopo atual do nó
	SymbolRef interface{} // Referência do símbolo (se aplicável)
}
