# Detecção de Código Duplicado Global

## Visão Geral

O ARIT implementa um analisador global de código duplicado que detecta funções com estrutura similar em múltiplos arquivos. Esta funcionalidade é especialmente útil para identificar padrões de código que foram copiados e colados com pequenas variações.

## Características

### 🎯 Detecção Cross-File
- Mantém estado global entre análises de múltiplos arquivos
- Detecta duplicações mesmo quando as funções estão em arquivos diferentes
- Compara estrutura de código, não apenas texto literal

### 🧠 Normalização Inteligente
- Substitui nomes de parâmetros por placeholders genéricos
- Normaliza nomes de funções para focar na estrutura
- Remove comentários e normaliza espaços em branco
- Detecta padrões similares mesmo com nomes diferentes

### ⚙️ Configurável
- **Threshold de Linhas**: Mínimo de 3 linhas (configurável)
- **Threshold de Tokens**: Mínimo de 15 tokens (configurável)
- Evita falsos positivos em funções muito pequenas

## Como Funciona

### 1. Extração de Código
```clojure
;; Função original
(defn process-user-data [user]
  (let [name (clojure.string/trim (:name user))
        email (clojure.string/lower-case (:email user))]
    {:processed-name name
     :processed-email email}))
```

### 2. Normalização
```clojure
;; Após normalização
(defn FUNC_NAME [PARAM]
  (let [name (clojure.string/trim (:name PARAM))
        email (clojure.string/lower-case (:email PARAM))]
    {:processed-name name
     :processed-email email}))
```

### 3. Hash e Comparação
- Gera hash MD5 do código normalizado
- Compara hashes para detectar duplicações
- Mantém registro de todas as ocorrências

## Exemplos de Detecção

### Exemplo 1: Funções Similares
```clojure
;; Arquivo 1
(defn process-user-data [user]
  (let [name (clojure.string/trim (:name user))
        email (clojure.string/lower-case (:email user))]
    {:processed-name name
     :processed-email email}))

;; Arquivo 2  
(defn process-customer-data [customer]
  (let [name (clojure.string/trim (:name customer))
        email (clojure.string/lower-case (:email customer))]
    {:processed-name name
     :processed-email email}))
```

**Resultado**:
```
[WARNING] duplicated-code-global: Duplicated code detected in function "process-user-data" 
(2 occurrences, 4 lines). Also found in: file2.clj:process-customer-data
```

### Exemplo 2: Múltiplas Duplicações
```clojure
;; Três funções com estrutura similar
(defn calculate-total [items] ...)
(defn compute-amount [values] ...)  
(defn sum-elements [data] ...)
```

**Resultado**:
```
[WARNING] duplicated-code-global: Duplicated code detected in function "compute-amount" 
(3 occurrences, 5 lines). Also found in: file1.clj:calculate-total, file3.clj:sum-elements
```

## Configuração

### Thresholds Padrão
```go
minLines:   3   // Mínimo de 3 linhas
minTokens:  15  // Mínimo de 15 tokens
```

### Personalização
Os thresholds podem ser ajustados no código fonte em `internal/rules/duplicated_code_global.go`:

```go
func GetGlobalDuplicatedCodeAnalyzer() *GlobalDuplicatedCodeAnalyzer {
    if globalAnalyzer == nil {
        globalAnalyzer = &GlobalDuplicatedCodeAnalyzer{
            codeBlocks: make(map[string][]CodeBlockInfo),
            minLines:   3,  // Ajuste aqui
            minTokens:  15, // Ajuste aqui
        }
    }
    return globalAnalyzer
}
```

## Limitações

### Escopo Atual
- Detecta apenas funções (`defn`)
- Não detecta duplicação em macros ou outras formas
- Normalização é baseada em padrões pré-definidos

### Falsos Positivos/Negativos
- **Falsos Positivos**: Funções legitimamente similares (ex: getters/setters)
- **Falsos Negativos**: Código duplicado com estrutura muito diferente

## Implementação Técnica

### Arquitetura
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   AnalyzeFile   │───▶│  GlobalAnalyzer  │───▶│   Findings      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   CodeBlocks     │
                       │   (Hash Map)     │
                       └──────────────────┘
```

### Componentes Principais
- **GlobalDuplicatedCodeAnalyzer**: Singleton que mantém estado global
- **CodeBlockInfo**: Estrutura com informações do bloco de código
- **Normalização**: Funções para padronizar código antes da comparação
- **Hash**: MD5 do código normalizado para comparação eficiente

## Integração

### No Pipeline de Análise
A detecção de código duplicado é executada automaticamente durante a análise:

```go
// Em analyzer.go
duplicatedFindings := globalDuplicatedAnalyzer.AnalyzeTree(tree, richRoots, filepath)
concreteFindings = append(concreteFindings, duplicatedFindings...)
```

### Reset Entre Execuções
Para análises independentes, o estado pode ser resetado:

```go
globalAnalyzer.Reset()
```

## Próximos Passos

### Melhorias Planejadas
1. **Configuração Externa**: Permitir configuração via arquivo
2. **Mais Formas**: Detectar duplicação em `defmacro`, `let`, etc.
3. **Normalização Avançada**: Usar AST para normalização mais precisa
4. **Métricas**: Adicionar métricas de similaridade
5. **Exclusões**: Permitir exclusão de padrões específicos

### Contribuições
Para contribuir com melhorias na detecção de código duplicado:
1. Adicione novos padrões de normalização
2. Melhore os thresholds baseado em dados reais
3. Implemente detecção para outras formas além de `defn`
4. Adicione testes para casos específicos 