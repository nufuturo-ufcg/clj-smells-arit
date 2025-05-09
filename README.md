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

## Contribuindo

A DEFINIR

## Licença

A DEFINIR