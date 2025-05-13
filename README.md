# Arit - Analisador Estático para Clojure

`arit` é uma ferramenta de linha de comando para análise estática de código Clojure (.clj, .cljs, .cljc). Ele ajuda a identificar potenciais problemas e aplicar regras de estilo para manter a qualidade do código.

## Funcionalidades

*   Análise de múltiplos arquivos e diretórios Clojure.
*   Suporte para regras de análise configuráveis via `.arit.yaml`.
*   Geração de relatórios em formatos: texto (padrão), JSON, HTML e Markdown.
*   Processamento paralelo de arquivos.

## Instalação e Execução

Para construir o executável a partir do código-fonte, navegue até a raiz do projeto e execute:

```bash
go build -o arit_analyzer ./cmd/arit
```

Isso criará um executável chamado `arit_analyzer` (ou `arit_analyzer.exe` no Windows) no diretório atual.

### Como Usar

Após a compilação, você pode executar `arit_analyzer` a partir da linha de comando:

```bash
# Analisar um arquivo específico
./arit_analyzer caminho/para/seu/arquivo.clj

# Analisar múltiplos arquivos
./arit_analyzer arquivo1.clj arquivo2.cljs

# Analisar todos os arquivos Clojure em um diretório (e subdiretórios)
./arit_analyzer caminho/para/seu/projeto/

# Especificar o formato de saída (por exemplo, HTML, redirecionando para um arquivo)
./arit_analyzer --format=html caminho/para/seu/projeto/ > report.html
```

Alternativamente, se você tiver Go configurado corretamente e o diretório `$GOPATH/bin` (ou `$HOME/go/bin`) no seu PATH, você pode instalar o `arit` globalmente com:

```bash
go install github.com/thlaurentino/arit/cmd/arit@latest
```
E então executar simplesmente com `arit ...`.

## Configuração

`arit` procura por um arquivo de configuração chamado `.arit.yaml` na raiz do projeto ou em diretórios pais para personalizar as regras de análise.

Exemplo de `.arit.yaml`:

```yaml
# rules:
#   nome-da-regra:
#     enabled: true
#     config_especifica: valor
```

## Formatos de Saída

A flag `--format` controla o formato do relatório:

*   `text` (padrão)
*   `json`
*   `html`
*   `markdown`

## Estrutura do Projeto

*   `cmd/arit/`: Ponto de entrada da aplicação (`main.go`).
*   `internal/`: Contém os pacotes `analyzer`, `config`, `reader`, `reporter`, `rules`.
*   `go.mod`, `go.sum`: Dependências do Go.
*   `.gitignore`: Arquivos ignorados pelo Git.
*   `README.md`: Este arquivo.

## Lista de Smells 

### Code Smells Tradicionais

*   [ ] **Duplicated Code:** Regra existente (`duplicated_code.go`), mas a implementação atual é um *placeholder*.
*   [x] **Long Function:** Implementada (`internal/rules/long_function.go`).
*   [x] **Long Parameter List:** Implementada (`internal/rules/long_parameter_list.go`).
*   [x] **Divergent Change:** Implementada (`internal/rules/divergent_change.go`).
*   [ ] **Shotgun Surgery:** Não implementada.
*   [x] **Primitive Obsession:** Implementada (`internal/rules/primitive_obsession.go`).
*   [x] **Message Chains:** Implementada (`internal/rules/message_chains.go`).
*   [x] **Middle Man:** Implementada (`internal/rules/middle_man.go`).
*   [ ] **Inappropriate Intimacy:** Não implementada.
*   [x] **Comments:** Implementada (`internal/rules/comments.go`).
*   [ ] **Mixed paradigms:** Não implementada.
*   [ ] **Library locker:** Não implementada.
*   **Pode ser necessário criar "sub smells" (Analisar)***
*   [ ] **Data Class:** Não implementada (Ver `Primitive Obsession`).
*   [ ] **Feature Envy:** Não implementada (Ver `Inappropriate Intimacy` / `Middle Man`).
*   [ ] **Large Class:** Não implementada (Ver `Long Function` / `Divergent Change`).

### Code Smells Específicos do Clojure

*   [x] **Overuse of high-order functions:** Implementada (`internal/rules/overuse_of_high_order_functions.go`).
*   [x] **Trivial lambda:** Implementada (`internal/rules/trivial_lambda.go`).
*   [x] **Deeply-nested call stacks:** Implementada (`internal/rules/deeply_nested.go`).
*   [x] **Inappropriate Collection:** Implementada (`internal/rules/inappropriate_collection.go`, também `linear_collection_scan.go`).
*   [x] **Underutilizing clojure features:** Implementada (`internal/rules/underutilizing_features.go`).
*   [ ] **Premature optimization:** Não implementada.
*   [x] **Lazy side effects:** Implementada (`internal/rules/lazy_side_effects.go`).
*   [x] **Immutability violation:** Implementada (`internal/rules/immutability_violation.go`).
*   [x] **External data coupling:** Não implementada.
*   [x] **Inefficient Filtering:** Implementada (`internal/rules/inefficient_filtering.go`).
*   [x] **Overabstracted Composition:** Implementada (`internal/rules/overabstracted_composition.go`).
*   [x] **Unnecessary Abstraction:** Implementada (Verificar "agressividade" da regra (`internal/rules/unnecessary_abstraction.go`).
*   [x] **Potentially Inefficient Generator:** Implementada (`internal/rules/potentially_inefficient_generator.go`). (Sub-regra de `Inefficient Filtering`).
*   [x] **String Map Keys:** Implementada (`internal/rules/string_map_keys.go`). (Não listada explicitamente no documento, mas implementada. Smell de teste).

**Legenda:**

*   `[x]` - Regra Implementada
*   `[ ]` - Regra Não Implementada (ou implementação incompleta/placeholder)

## Contribuindo

A DEFINIR

## Licença

A DEFINIR